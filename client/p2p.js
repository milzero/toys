'use strict';

let localVideo;
let localStream;
let remoteVideo;
let peerConnection;
let uuid;
let roomId;
let serverConnection;
let audioInputSelect;
let audioOutputSelect;
let videoSelect;
let selectors;
let startBiterate;

function attachSinkId(element, sinkId) {
    if (typeof element.sinkId !== "undefined") {
        element
            .setSinkId(sinkId)
            .then(() => {
                console.log(`Success, audio output device attached: ${sinkId}`);
            })
            .catch((error) => {
                let errorMessage = error;
                if (error.name === "SecurityError") {
                    errorMessage = `You need to use HTTPS for selecting audio output device: ${error}`;
                }
                console.error(errorMessage);
                // Jump back to first output device in the list as it's the default.
                audioOutputSelect.selectedIndex = 0;
            });
    } else {
        console.warn("Browser does not support output device selection.");
    }
}

function changeAudioDestination() {
    const audioDestination = audioOutputSelect.value;
    attachSinkId(localVideo, audioDestination);
}

function handleError(error) {
    console.log(
        "error: ",
        error.message,
        error.name
    );
}

var peerConnectionConfig = {
    iceServers: [
        {
            urls: "turn:101.35.111.10:3478",
            username: "daozhao",
            credential: "12345",
        },
    ],
};

function join() {
    let data = JSON.stringify({
        video: true,
        audio: true,
    });
    serverConnection.send(
        JSON.stringify({
            event: "join",
            room_id: roomId,
            user_id: uuid,
            data: data,
        })
    );
}

function gotDevices(deviceInfos) {
    const values = selectors.map((select) => select.value);
    selectors.forEach((select) => {
        while (select.firstChild) {
            select.removeChild(select.firstChild);
        }
    });

    for (let i = 0; i !== deviceInfos.length; ++i) {
        const deviceInfo = deviceInfos[i];
        const option = document.createElement("option");
        option.value = deviceInfo.deviceId;
        if (deviceInfo.kind === "audioinput") {
            option.text =
                deviceInfo.label || `microphone ${audioInputSelect.length + 1}`;
            audioInputSelect.appendChild(option);
        } else if (deviceInfo.kind === "audiooutput") {
            option.text =
                deviceInfo.label || `speaker ${audioOutputSelect.length + 1}`;
            audioOutputSelect.appendChild(option);
        } else if (deviceInfo.kind === "videoinput") {
            option.text = deviceInfo.label || `camera ${videoSelect.length + 1}`;
            videoSelect.appendChild(option);
        } else {
            console.log("some other kind of source/device: ", deviceInfo);
        }
    }

    selectors.forEach((select, selectorIndex) => {
        if (
            Array.prototype.slice
                .call(select.childNodes)
                .some((n) => n.value === values[selectorIndex])
        ) {
            select.value = values[selectorIndex];
        }
    });

}


function call() {
    peerConnection.createOffer()
        .then((offer) => peerConnection.setLocalDescription(offer))
        .then(() => {
            serverConnection.send(JSON.stringify({
                event: "offer",
                room_id: roomId,
                user_id: uuid,
                sdp: peerConnection.localDescription,
            }));
        })
        .catch((reason) => {
            console.log("create offer  faile :" + reason);
        });
}

// 应答会话描述，生成answer
function answer() {
    peerConnection.createOffer()
        .then((offer) => peerConnection.setLocalDescription(offer))
        .then(() => {
            serverConnection.send(JSON.stringify({
                event: "answer",
                room_id: roomId,
                user_id: uuid,
                sdp: peerConnection.localDescription,
            }));
            console.log("answer send");
        })
        .catch((reason) => {
            console.log("create offer  faile :" + reason);
        });
}


function setMediaSource() {
    if (window.stream) {
        window.stream.getTracks().forEach((track) => {
            track.stop();
        });
    }
    const audioSource = audioInputSelect.value;
    const videoSource = videoSelect.value;
    const constraints = {
        audio: { deviceId: audioSource ? { exact: audioSource } : undefined },
        video: { deviceId: videoSource ? { exact: videoSource } : undefined, width: { exact: 640 }, height: { exact: 480 }, },
    };

    navigator.mediaDevices
        .getUserMedia(constraints)
        .then(getUserMediaSuccess)
        .then(gotDevices)
        .catch(handleError);
}

function connect() {
    serverConnection = new WebSocket("ws://127.0.0.1:18080/webrtc/p2p");
    serverConnection.onmessage = gotMessageFromServer;
}

function pageReady() {
    audioInputSelect = document.querySelector("select#audioSource");
    audioOutputSelect = document.querySelector("select#audioOutput");
    videoSelect = document.querySelector("select#videoSource");
    selectors = [audioInputSelect, audioOutputSelect, videoSelect];
    localVideo = document.getElementById("localVideo");
    remoteVideo = document.getElementById("remoteVideo");
    uuid = document.getElementById("uid").value;
    roomId = document.getElementById("roomId").value;
    startBiterate = 2000;

    navigator.mediaDevices.enumerateDevices().then(gotDevices).catch(handleError);
    setMediaSource();
    connect();
    peerConnection = new RTCPeerConnection(peerConnectionConfig);
    peerConnection.onicecandidate = gotIceCandidate;
    peerConnection.ontrack = gotRemoteStream;
    audioInputSelect.onchange = setMediaSource;
    audioOutputSelect.onchange = changeAudioDestination;
    videoSelect.onchange = setMediaSource;
}

function getUserMediaSuccess(stream) {
    localStream = stream;
    localVideo.srcObject = stream;
    return navigator.mediaDevices.enumerateDevices();
}

function gotMessageFromServer(message) {
    let signal = JSON.parse(message.data);
    console.log(signal);
    if (signal.uuid == uuid) return;
    if (signal.event == "offer") {
        peerConnection
            .setRemoteDescription(new RTCSessionDescription(signal.sdp))
            .then(function () {
                if (signal.sdp.type == "offer") {
                    peerConnection
                        .createAnswer()
                        .then(createdDescription)
                        .catch(errorHandler);
                }
            })
            .catch(errorHandler);
    } else if (signal.event === 'answer') {
        peerConnection.setRemoteDescription(new RTCSessionDescription(signal.sdp)).then(function () {
            console.log("got answer and  setRemoteDescription");
        })
            .catch(errorHandler);
    } else if (signal.event == "candidate") {
        let ice = JSON.parse(signal.data);
        peerConnection
            .addIceCandidate(new RTCIceCandidate(ice))
            .catch(errorHandler);
    }
}

function gotIceCandidate(event) {
    console.log("got ice candidate" , event);
    if (event.candidate != null) {
        let ice = JSON.stringify(event.candidate);
        serverConnection.send(
            JSON.stringify({
                event: "candidate",
                room_id: roomId,
                user_id: uuid,
                data: ice,
            })
        );
    }
}

function createdDescription(description) {
    console.log("created description");
    peerConnection
        .setLocalDescription(description)
        .then(function () {
            serverConnection.send(
                JSON.stringify({
                    event: "answer",
                    room_id: roomId,
                    user_id: uuid,
                    sdp: peerConnection.localDescription,
                })
            );
            console.log("send answer description");
        })
        .catch(errorHandler);
}

function gotRemoteStream(event) {
    console.log("got remote stream");
    if (event.track.kind === "audio") {
        return;
    }

    let el = document.createElement(event.track.kind);
    el.srcObject = event.streams[0];
    el.autoplay = true;
    el.controls = true;
    document.getElementById("remoteVideos").appendChild(el);

    event.track.onmute = function (event) {
        el.play();
    };

    event.streams[0].onremovetrack = ({ track }) => {
        if (el.parentNode) {
            el.parentNode.removeChild(el);
        }
    };
}

function errorHandler(error) {
    console.log(error);
}

