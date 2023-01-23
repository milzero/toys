
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