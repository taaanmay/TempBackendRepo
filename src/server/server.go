package main

/*
	fatalf and fatalln may have to be replaced from everything except for authenticateServer as it stops the program

	listenAndServe in wrong place, no ctx, post used instead of user, NewUser instead of NewRef and should be = not :=
*/

import (
	"log"
	"firebase.google.com/go"
	"firebase.google.com/go/auth"
	"net/http" // UNCOMMENT this when using http package
	"encoding/json"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"fmt"
	"encoding/json"
)

var app *firebase.App

//specify user interface
type User struct {
    rank int "json:rank"
    username string "json:username"
    score float64 "json:score"
}

type requestMessage struct {
	auth string		"json:auth"
}


type endConversationRequest struct{
	auth string  "json: auth"
	guess int    "json: guess"
}

type newUser struct {
    username string "json:  username" 
    score float64 "json: score"
}


func main() {
	app := authenticateServer()

	// database client
	dbClient, err := app.Database(context.Background())
	if err != nil {
		log.Fatalln("Error initializing database client:", err)
	}

	//TODO just test dataset output, remove later
	// context.Background just creates empty context that we can pass if we dont need this tool
	ref := dbClient.NewRef("")
	var data map[string]interface{}
	if err := ref.Get(context.Background(), &data); err != nil {
		log.Fatalln("Error reading from database:", err)
	}
	fmt.Println(data)

	go func() {
		log.Fatal(http.ListenAndServe("localhost:420/api/conversation/receive/:cid", http.HandlerFunc(receiveMessageHandler)))
	} ()

	go func() {
		log.Fatal(http.ListenAndServe("localhost:421/api/leaderboards", http.HandlerFunc(leaderboardHandler)))
	} ()

	log.Fatal(http.ListenAndServe("localhost:430", nil))
}

// This method should be called on initialization
// It connects our server as an admin and returns the app instance
func authenticateServer() *firebase.App {
	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL: "https://turing-game-e5059.firebaseio.com",
	}
	// Fetch the service account key JSON file contents
	opt := option.WithCredentialsFile("serverKeys.json")

	// Initialize the app with a service account, granting admin privileges
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		log.Fatalln("Error initializing app:", err)
	}
	return app
}

// Method verifies if incoming token is a valid and logged-in users token
func checkUserAuthentication(idToken string) (*auth.Token, error) {
	ctx := context.Background()
	client, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Fatalf("error verifying ID token: %v\n", err)
		return nil, err
	}

	log.Printf("Verified ID token: %v\n", token)
	return token, nil
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
  // Create a database client from App.
  client, err := app.Database(ctx)
  if err != nil {
          log.Fatalln("Error initializing database client:", err)
  }

  // Get a database reference to posts(? need update)
  ref := client.NewRef("?")

  // Read the data at the posts reference (this is a blocking operation)
  var user User
  if err := ref.Get(ctx, &user); err != nil {
          log.Fatalln("Error reading value:", err)
  }

  //arrange the users in database by score
  ref = client.NewRef("leaderboards")

  result, err := ref.OrderByValue().GetOrdered(ctx)
  if err != nil {
        log.Fatalln("Error querying database:", err)
  }
}

// handler for when a call to a url is reached to get the messages
func receiveMessageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cid := r.URL.Path[len("/api/conversation/receive/"):]
	var receiveMessageRequest requestMessage

	err := json.NewDecoder(r.Body).Decode(&receiveMessageRequest)
	// places all the data from the body into receiveMessageRequest and puts an error into err
	if err != nil {
		log.Printf("JSON decoding failed %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// now we have the cid and the user auth

	id, err := checkUserAuthentication(receiveMessageRequest.auth)	// validate the user  is real
	if err != nil {
		log.Printf("Error authenticating user %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// user is authenticated and the we have cid and user auth, now we can look to get all the data
	client, err := app.Database(ctx)
	if err != nil {
		log.Fatalln("Error initializing database client: ", err)
	}

	// need to use decrypted data to go into firebase, find the correct chat room and then collect
	// all of the messages from that one person in that chat room

	// have:
	//					 cid
	// 					 userId
	//					 database client
	// want to: get database reference, go into chatRoom with corresponding cid, collect all of the
	// messages with the userId that we have, put them into w and then return

	// ref to the chatRooms, query on it to find the chat room we want
	chatRoomsRef := client.NewRef("chatRooms/chatRoom")

	// the chat room we want the messages from
	chatRoomRef, err := chatRoomsRef.OrderByKey().EqualTo(cid).GetOrdered(ctx)

	messagesRef, err := chatRoomRef.OrderByChild("messages/id").EqualTo(id.UID).GetOrdered(ctx)

}



func endConversationHandler(w http.ResponseWriter, r *http.Request){

ctx := context.Background()
//Authenticate User
var endReq endConversationRequest
err := json.NewDecoder(r.Body).Decode(&endReq)
auth:=endReq.auth
userToken, err := checkUserAuthentication(auth)
if err!=nil {
	log.Fatalf("error verifying ID token: %v\n", err)
} 


//get a reference to the chat rooms section of the database
//cid:= r.URL.Path[1:]
cid:= r.URL.Path[len("/api/conversation/end/"):]	
client, err := app.Database(ctx)

ref := client.NewRef("availableGames/chatRoomId")
//results,err := ref.OrderByKey().EqualTo(cid).GetOrdered(ctx)
if err != nil{
	log.Fatalln("Error querying database:", err)
}

//make this chatroom complete
//roomStatus := results.Get()
address := "availableGames/" + cid
ref = client.NewRef(address)
err = ref.Set(ctx,"complete")
if err != nil{
	log.Fatalln("Error setting value as complete:", err)
}

// Submit guess

userRef := client.NewRef("leaderboards/user")
_,err = userRef.OrderByKey().EqualTo(userToken.UID).GetOrdered(ctx)
if err!= nil{

	if _, err := userRef.Push(ctx, &newUser{
	username: userToken.UID,
	score: 1,
	});
	err != nil {
	log.Fatalln("Error pushing child node:", err)

	}
	return
}
//Getting the score of the user
var nUser newUser
userAdd := "leaderboards/" + userToken.UID
userRef = client.NewRef(userAdd)
 if err = userRef.Get(ctx,&nUser); err!= nil{
 	log.Fatalln("Error getting value:", err)}

//Setting the score of the user
userAdd = "leaderboards/" + userToken.UID + "/score"
ref = client.NewRef(userAdd)
userScore := nUser.score + 1;
err = ref.Set(ctx,userScore)
if err != nil{
	log.Fatalln("Error setting value", err)
}



}
