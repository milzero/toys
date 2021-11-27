var localVideo;
var localStream;
var remoteVideo;
var peerConnection;
var uuid;
var roomId;
var serverConnection;

var peerConnectionConfig = {
    iceServers: [{
        urls: "turn:101.35.111.10:3478",
        username: "daozhao",
        credential: "12345",
    }, ],
};

function pageReady() {
    localVideo = document.getElementById("localVideo");
    remoteVideo = document.getElementById("remoteVideo");

    uuid = document.getElementById("uid").value;
    roomId = document.getElementById("roomId").value;

    serverConnection = new WebSocket(
        "ws://127.0.0.1:8080/"
    );
    serverConnection.onmessage = gotMessageFromServer;


    var constraints = {
        video: true,
        audio: true,
    };


    if (navigator.mediaDevices.getUserMedia) {
        navigator.mediaDevices
            .getUserMedia(constraints)
            .then(getUserMediaSuccess)
            .catch(errorHandler);
    } else {
        alert("Your browser does not support getUserMedia API");
    }

}

function getUserMediaSuccess(stream) {
    localStream = stream;
    localVideo.srcObject = stream;
}

function publish(){
    serverConnection.send(
        JSON.stringify({
            event: "publish",
            room_id: uuid,
            user_id: roomId,
            data: '{"video": "true","audio": "true",}'
        })
    );     
}

function join(){

    var data = JSON.stringify({
        video: true,
        audio: true,
    });
    serverConnection.send(
        JSON.stringify({
            event: "join",
            room_id: uuid,
            user_id: roomId,
            data: data,
        })
    );     
}

function unPublish(){
    serverConnection.send(
        JSON.stringify({
            event: "unPublish",
            room_id: uuid,
            user_id: roomId,
            data: {
                video: true,
                audio: true,
            }
        })
    );     
}

function subscribe(){
    serverConnection.send(
        JSON.stringify({
            event: "unPublish",
            room_id: uuid,
            user_id: roomId,
            users: [],
            data: {
                video: true,
                audio: true,
            }
        })
    );     
}

function unSubscribe(){
    serverConnection.send(
        JSON.stringify({
            event: "unPublish",
            room_id: uuid,
            user_id: roomId,
            users: [],
            data: {
                video: true,
                audio: true,
            }
        })
    );     
}

function start(isCaller) {
    peerConnection = new RTCPeerConnection(peerConnectionConfig);

    peerConnection.onicecandidate = gotIceCandidate;
    peerConnection.ontrack = gotRemoteStream;
    peerConnection.addStream(localStream);
}

function gotMessageFromServer(message) {
    if (!peerConnection) start(false);
    var signal = JSON.parse(message.data);
    console.log(signal);

    // Ignore messages from ourself
    if (signal.uuid == uuid) return;
    var event = signal.data;

    if (signal.event == "offer") {
        offer = JSON.parse(signal.data);
        peerConnection
            .setRemoteDescription(new RTCSessionDescription(offer))
            .then(function () {
                // Only create answers in response to offers
                if (offer.type == "offer") {
                    peerConnection
                        .createAnswer()
                        .then(createdDescription)
                        .catch(errorHandler);
                }
            })
            .catch(errorHandler);
    } else if (signal.event == "candidate") {
        var ice =  JSON.parse(signal.data);
        peerConnection
            .addIceCandidate(new RTCIceCandidate(ice))
            .catch(errorHandler);
    }
}

function gotIceCandidate(event) {
    if (event.candidate != null) {
        var ice = JSON.stringify(event.candidate);
        console.log(
            JSON.stringify({
                event: "candidate",
                room_id: uuid,
                user_id: roomId,
                data: ice
            })
        );
        serverConnection.send(
            JSON.stringify({
                event: "candidate",
                room_id: uuid,
                user_id: roomId,
                data: ice
            })
        );
    }
}

function createdDescription(description) {
    console.log("got description");
    peerConnection
        .setLocalDescription(description)
        .then(function () {
            var sdp   = JSON.stringify(peerConnection.localDescription); 
            console.log(
                JSON.stringify({
                    event: "answer",
                    room_id: uuid,
                    user_id: roomId,
                    data: sdp
                })
            );

            serverConnection.send(
                JSON.stringify({
                    event: "answer",
                    room_id: uuid,
                    user_id: roomId,
                    data: sdp
                })
            );
        
        })
        .catch(errorHandler);
}

function gotRemoteStream(event) {
    console.log("got remote stream");
    remoteVideo.srcObject = event.streams[0];
}

function errorHandler(error) {
    console.log(error);
}