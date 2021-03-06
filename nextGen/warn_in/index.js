/**************************************************************
 *
 *  Copyright (c) 2018 Public Broadcasting Service
 *  Contact: <warn@pbs.org>
 *  All Rights Reserved.
 *
 *  Version 1.23 12/18/2018
 *
 *************************************************************/

"use strict"

const moment = require('moment')
const xml2js = require('xml2js')
const mysql = require('mysql2/promise')
const atob = require('atob')

var dbTimeFormat = "YYYY-MM-DD HH:mm:ssZ"

var db_config = {
        connectionLimit     : 200,
        host                : 'warn.cluster-czmkuxdhszlq.us-west-2.rds.amazonaws.com',
        port                : 3306,
        user                : 'warn',
        password            : 'warnwarn',
        waitForConnections  : true,
        connectTimeout      : 20000,
        queueLimit          : 0
    }
    
var pool = mysql.createPool(db_config)

// on POST call
exports.handler = async (event, context, callback) => {
    context.callbackWaitsForEmptyEventLoop = false  // asynchronize the handler callback
    var now = moment().format(dbTimeFormat)
    await updateHeartbeat(now)
    var message = event.body
    // if it's a labeled heartbeat message, short-cut out
    //console.log("Message:", message)
    if (message == 'heartbeat') {
        await callback(null, {statusCode:"200", body:'heartbeat'}) 
        return
    } else {
        // send message to DB (dupes will be ignored)
        var [uid, xml, alertExpires, callback] = await procAlert(context, message, callback)
        var [status, rsp] = await postAlert(now, uid, xml, alertExpires, callback)
        rsp = now + rsp
        await callback(null, {statusCode:status, body:rsp})
    }
}


// update warn.heartbeat.latest in DB
async function updateHeartbeat(now) {
    var sql = "UPDATE warn.heartbeat SET latest=? WHERE Id=1"
    try {
        await pool.execute(sql, [now])
    } catch(e) {
        console.log("updateHeartbeat pool.execute Error:", e)
    }
}

async function procAlert(context, xml, callback) {
    var uid, alertExpires, ns
    xml2js.parseString(xml, function (err, result) {
        if (err) {
            console.log("xml2js Error",err)
            return
        }
        // extract XML namespace
        ns = ""
        if (typeof result.alert.$.xmlns != 'undefined') {
            ns = result.alert.$.xmlns // the XML default namespace
        }
        // if it's a CAP message, post the new alert to DB
        if (ns == "urn:oasis:names:tc:emergency:cap:1.2") {
            var alert = result.alert
            uid = alert.identifier + "," + alert.sender + "," + alert.sent // per CAP spec
            // If Update or Cancel, mark each of the referenced messages as replaced
            if (alert.msgType == "Cancel" || alert.msgType == "Update") {
                if (typeof(alert.references) == "string") {
                    var references = alert.references.split(" ")
                    for (var i in references) {
                        var target = references[i]
                        // fire-forget an sql call to update the target record
                        replaced(target, uid)
                    }
                }
            }
            // Now extract the latest expires time across all Infos
            alertExpires = ""
            if (typeof alert != 'undefined' && typeof alert.info != 'undefined') {
                for (var info of alert.info) {
                    if (info.expires > alertExpires) {
                        alertExpires = info.expires
                    }
                }
            }
        }
    }) 
    return [uid, xml, alertExpires, callback]
}

async function postAlert(now, uid, xml, expires, callback) {
    var sql = "INSERT INTO warn.alerts (uid, xml, expires, received) VALUES (?,?,?,?)"
    var status, rsp = ""
    try {
        expires = moment(expires[0]).format(dbTimeFormat)
    } catch (e) {
        console.log("postAlert expires =", expires, "for", uid)
    }
    try {
        await pool.execute(sql, [uid, xml, expires, now])
        status = "200"
        rsp = " ADD " + uid
    } catch(e) {
        if (e.message.includes("Duplicate entry")) {
            status = "200"
            rsp = " DUP " + uid
        } else {
            status = "500"
            rsp = " ERROR " + uid + " " + e
        }
    }
    return [status, rsp]  // status and response string will be returned to sender
}

async function replaced(target, uid) {
    var sql = 'UPDATE warn.alerts SET replacedBy = ? WHERE uid = ?'
    try {
        pool.execute(sql, [uid, target])
        console.log("REPLACEMENT for ", target)
    } catch (e) {
        console.log("replaced() Error", e)
    }
}
