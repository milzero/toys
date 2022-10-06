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
    "navigator.MediaDevices.getUserMedia error: ",
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


function gotDevices(deviceInfos) {
  // Handles being called several times to update labels. Preserve values.
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
      console.log("Some other kind of source/device: ", deviceInfo);
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
    video: { deviceId: videoSource ? { exact: videoSource } : undefined },
  };
  navigator.mediaDevices
    .getUserMedia(constraints)
    .then(getUserMediaSuccess)
    .then(gotDevices)
    .catch(handleError);
}

function connect() {
  serverConnection = new WebSocket("ws://127.0.0.1:18080/webrtc");
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

  navigator.mediaDevices.enumerateDevices().then(gotDevices).catch(handleError);
  setMediaSource();
  connect();
  audioInputSelect.onchange = setMediaSource;
  audioOutputSelect.onchange = changeAudioDestination;
  videoSelect.onchange = setMediaSource;
}

function getUserMediaSuccess(stream) {
  localStream = stream;
  localVideo.srcObject = stream;
  // window.setInterval(getStatus, 1000 * 10);
  return navigator.mediaDevices.enumerateDevices();
}

function publish() {
  serverConnection.send(
    JSON.stringify({
      event: "publish",
      room_id: roomId,
      user_id: uuid,
      data: '{"video": "true","audio": "true",}',
    })
  );
}

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

function unPublish() {
  serverConnection.send(
    JSON.stringify({
      event: "unPublish",
      room_id: roomId,
      user_id: uuid,
      data: {
        video: true,
        audio: true,
      },
    })
  );
}

function subscribe() {
  serverConnection.send(
    JSON.stringify({
      event: "unPublish",
      room_id: roomId,
      user_id: uuid,
      users: [],
      data: {
        video: true,
        audio: true,
      },
    })
  );
}

function unSubscribe() {
  serverConnection.send(
    JSON.stringify({
      event: "unPublish",
      room_id: roomId,
      user_id: uuid,
      users: [],
      data: {
        video: true,
        audio: true,
      },
    })
  );
}

function start() {
  peerConnection = new RTCPeerConnection(peerConnectionConfig);
  peerConnection.onicecandidate = gotIceCandidate;
  peerConnection.ontrack = gotRemoteStream;
  peerConnection.addStream(localStream);
}

function gotMessageFromServer(message) {
  if (!peerConnection) start();
  let signal = JSON.parse(message.data);
  console.log(signal);

  // Ignore messages from ourself
  if (signal.uuid == uuid) return;
  let event = signal.data;

  if (signal.event == "offer") {
    let offer = JSON.parse(signal.data);
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
    let ice = JSON.parse(signal.data);
    peerConnection
      .addIceCandidate(new RTCIceCandidate(ice))
      .catch(errorHandler);
  }
}

function gotIceCandidate(event) {
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
  console.log("got description");
  peerConnection
    .setLocalDescription(description)
    .then(function () {
      let sdp = JSON.stringify(peerConnection.localDescription);
      serverConnection.send(
        JSON.stringify({
          event: "answer",
          room_id: roomId,
          user_id: uuid,
          data: sdp,
        })
      );
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

function getStatus() {
  if (peerConnection == null) {
    console.error("peer connect is null");
    return;
  }

  let senders = peerConnection.getSenders();
  console.log("size of senders is ", senders.length);
  peerConnection.getStats().then(function () {
    (async () => {
      let statsOutput = "";
      const report = await peerConnection.getStats();
      for (let dictionary of report.values()) {
        console.log(dictionary);
        statsOutput = '<p>' + dictionary.type + '</p>';
        statsOutput = statsOutput + '<p>' + dictionary.type + '</p>';
        statsOutput = statsOutput + '<p>' + '  timestamp: ' + dictionary.timestamp + '</p>';
        Object.keys(dictionary).forEach(key => {
            if (key != 'type' && key != 'id' && key != 'timestamp') {
                statsOutput = statsOutput + '<p>' + '  ' + key + ': ' + dictionary[key] + '</p>'
            }
        });
      }

      document.getElementById(".stats-box").innerHTML = statsOutput;
      console.log(statsOutput);
    })();
  });
}
