/**
 * Created by InsZVA on 2017/2/4.
 */

/**
 * ConnMaster handlers all message from the websocket to the server,
 * and dispatch them to all client connections
 * @param {string} url
 * @constructor
 */
function ConnMaster(url) {
    this.onopen = this.onerror = this.onclose = null;

    this.ws = {url: url};

    this._clientConns = [];
    this.onmessage = null;
    this.onunkownmessage = null;

    /** cache a client message if the clientconn not found
        to make p2p connect fast
     */
    this.cachedClientMessage = [];
}

ConnMaster.prototype.connect = function() {

    this.ws = new WebSocket(this.ws.url);
    this.ws.onmessage = this.dispatchMsg.bind(this);
    this.ws.onopen = this.onopen;
    this.ws.onerror = this.onerror;
    this.ws.onclose = this.onclose;
};

/**
 * send a object via websocket using json
 * @param {Object} data
 */
ConnMaster.prototype.send = function (data) {
    var str = JSON.stringify(data);
    this.ws.send(str);
};


ConnMaster.prototype.dispatchMsg = function(e) {
    var data;
    try {
        data = JSON.parse(e.data);
    } catch (exception) {
        if (this.onunkownmessage)
            return this.onunkownmessage(e);
        return null;
    }

    if (data.type && data.type != "forward") {
        return this.onmessage(e);
    }

    if (data.srcId) {
        if (this._clientConns[data.srcId] && this._clientConns[data.srcId].onmessage) {
            return this._clientConns[data.srcId].onmessage(e);
        } else {
            console.log("cached");
            if (!this.cachedClientMessage)
                this.cachedClientMessage[data.srcId] = [];
            this.cachedClientMessage.push(e);
            if (this.onunkownmessage) {
                return this.onunkownmessage(e);
            } else {
                return null;
            }
        }
    } else {
        if (this.onunkownmessage)
            return this.onunkownmessage(e);
    }
};

/**
 * Create a client conn and register its message handler to Conn Master
 * !!!NOTE!!! should use this function rather directly ClientConn constructor
 * @param remoteId
 * @returns {ClientConn}
 */
ConnMaster.prototype.newClientConn = function(remoteId) {
    if (this._clientConns[remoteId])
        return this._clientConns[remoteId];
    var c = new ClientConn(this, remoteId);
    this._clientConns[remoteId] = c;
    return c;
};

/**
 * Client conn represents a connection to a client proxy by server
 * @param {ConnMaster} masterConn
 * @param {number} remoteId
 * @constructor
 */
function ClientConn(masterConn, remoteId) {
    this._masterConn = masterConn;
    this._remoteId = remoteId;
    // function(event)
    this.onmessage = null;
}

/**
 * JSON encode the data and send proxy by server to remote client
 * @param {Object} data
 */
ClientConn.prototype.send = function(data) {
    data.srcId = LocalClient.id;
    data.dstId = this._remoteId;
    data.type = "forward";
    this._masterConn.send(data);
};