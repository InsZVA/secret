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
        this.dc = ev.channel;
        this.state = "transferring";
    }.bind(this);
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
        if (data.candidate != "")
            this.pc.addIceCandidate(new RTCIceCandidate(data.candidate))
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
 * @param {Event} e
 */
Output.prototype.send = function(e) {
    var data = new Uint8Array(e.data);
    this.dc.send(data);
};