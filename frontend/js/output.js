/**
 * Created by InsZVA on 2017/2/18.
 */


function Output() {
    /**
     * close --beginP2P--> connecting --connected--> transferring --statistics--> [ready] --need--> running/reserved
     *                      |failed|                                |out time|            |not need|
     *                        close                                  timeout               released
     * @type {string}
     */
    this.state = "close";
    this.remote = null;
    // PeerConnection
    this.pc = new RTCPeerConnection(LocalClient.rtcConfig);
    // MasterConn
    this._masterConn = null;
    // ClientConn
    this.conn = null;
}

/**
 * bind a output to a client
 * @param {ConnMaster} masterConn
 * @param {string} clientId
 */
Output.prototype.bind = function (masterConn, clientId) {
    console.log(masterConn);
    this.remote = clientId;
    this._masterConn = masterConn;

    this.conn = masterConn.newClientConn(clientId);
    this.state = "connecting";
    this.pc.onicecandidate = function(event){
        this.conn.send({
            cmd: "icecandidate",
            candidate: event.candidate
        });
    }.bind(this);
    this.conn.onmessage = this._onmessage.bind(this);

    // If cached offer, directly use it
    if (masterConn.cachedClientMessage[clientId]) {
        console.log("use cached");
        if (masterConn.cachedClientMessage)
            for (var i = 0; i < masterConn.cachedClientMessage.length; i++) {
                var e = masterConn.cachedClientMessage[clientId][i];
                this.conn.onmessage(e);
            }
        masterConn.cachedClientMessage[clientId] = undefined;
    }

    this.pc.ondatachannel = function(ev) {
        console.log(ev);
        this.dc = ev.channel;
        this.dc.onopen = function() {
            this.fastload();
            this.state = "transferring";
        }.bind(this);
        this.dc.onclose = this.close.bind(this);
        this.dc.onmessage = function(e) {
            var msg;
            try {
                msg = JSON.parse(e.data);
            } catch (exception) {
                throw exception;
            }
            if (msg.reserved) {
                this.reserve();
            }
        }.bind(this);
    }.bind(this);
};

Output.prototype.fastload = function() {
    if (!this.dc) return;
    var running = LocalClient.runningInput();
    if (running == -1) return;
    this.send(LocalClient.initmsg[0].raw);
    this.send(LocalClient.initmsg[1].raw);
    var fastloads = [];

    for (var j = 0; j < 2; j++)
        for (var i = 0; i < LocalClient.bufferqueue[j]._handlerqueue.length; i++) {
            fastloads.push(LocalClient.bufferqueue[j]._handlerqueue[i].raw);
        }
    while (fastloads.length > 0) {
        var index = parseInt(Math.random() * fastloads.length);
        this.send(fastloads[index]);
        fastloads = fastloads.slice(0, index).concat(
            fastloads.slice(index+1)
        );
    }
};

/**
 * on WebRTC signal message
 * @param {Event} e
 * @returns {null}
 * @private
 */
Output.prototype._onmessage = function(e) {
    var data = e.data;
    var msg;
    try {
        msg = JSON.parse(data);
    } catch(exception) {
        return null;
    }
    if (msg.cmd && msg.cmd == "icecandidate") {
        this.pc.addIceCandidate(new RTCIceCandidate(msg.candidate))
    } else if (msg.cmd && msg.cmd == "offer") {
        this.pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
        this.pc.createAnswer().then(function(answer) {
            return this.pc.setLocalDescription(answer);
        }.bind(this)).then(function() {
            this.conn.send({
                cmd: "answer",
                sdp: this.pc.localDescription
            });
        }.bind(this));
    }
};

/**
 * send a event by data-channel
 * @param {*} data
 */
Output.prototype.send = function(data) {
    this.dc.send(data);
};

/**
 * close the output and clean the resource
 */
Output.prototype.close = function() {
    console.log("close output");
    this.state = "close";
};

/**
 * change this output to reserved, this function will be called
 * when the peer send "reserve" message via datachannel.
 */
Output.prototype.reserve = function() {
    console.log("reserve output");
    this.state = "reserved";
};