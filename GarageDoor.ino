#include <ESP8266WiFi.h>
#include <WiFiClient.h>
#include <ESP8266WebServer.h>
#include <ESP8266mDNS.h>
#include <ESP8266HTTPClient.h>

// SSID details
const char* ssid     = "<YOUR_SSID_HERE>";     // CHANGE ME!
const char* password = "<YOUR_WIFI_PASSWORD>"; // CHANGE ME!

// HTTP Basic Auth Creds
const char* httpuser = "admin";    // CHANGE ME!
const char* httppass = "password"; // CHANGE ME!

// GPIO Pins that will be used
const int sensorClosed = D1;  // door sensor circuit (bottom)
const int sensorOpen   = D2;  // door sensor circuit (top)
const int relay        = D5;  // control garage door button

// mDNS name
const char* host = "garagedoor";

const char* refreshURL = "http://192.168.0.101:8180/refresh";

const int opened     = 0;
const int closed     = 1;
const int opening    = 2;
const int closing    = 3;
const int unknown    = 4;

const int btnPush    = HIGH;
const int btnRelease = LOW;

const unsigned long duration = 16000;

// tracking vars for door state
int           lastCState = unknown;
int           lastOState = unknown;
int           lastState = unknown;
unsigned long lastPress = 0;

ESP8266WebServer server(80);

void setup() {
  Serial.begin(115200);
  
  // Init GPIO state
  pinMode(sensorClosed, INPUT);
  pinMode(sensorOpen, INPUT);
  digitalWrite(relay, btnRelease);
  pinMode(relay, OUTPUT);

  // Connect to Wi-Fi network with SSID and password
  Serial.print("Attempting WiFi connection to: ");
  Serial.println(ssid);
  WiFi.mode(WIFI_STA);
  WiFi.begin(ssid, password);
  Serial.println("");

  // Wait for connection
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.println("");
  Serial.print("Connected to ");
  Serial.println(ssid);
  Serial.print("IP address: ");
  Serial.println(WiFi.localIP());

  if (MDNS.begin(host)) {
    Serial.print("mDNS responder started with name: ");
    Serial.println(host);
  }

  server.on("/open", HTTP_POST, handleOpen);
  server.on("/close", HTTP_POST, handleClose);
  server.on("/press", HTTP_POST, handlePress);
  server.on("/", handleRoot);
  server.begin();
  Serial.println("HTTP server started");
}

void loop(void){
  if (stateChanged()) {
    handleStateChange();
  }
  server.handleClient();
}

bool stateChanged() {
  int cState = digitalRead(sensorClosed);
  int oState = digitalRead(sensorOpen);

  if (cState != lastCState || oState != lastOState) {
    return true;
  }
  return false;
}

void handleStateChange() {
  int prevCState = lastCState;
  int prevOState = lastOState;
  lastCState = digitalRead(sensorClosed);
  lastOState = digitalRead(sensorOpen);

  if (prevCState != lastCState) {
    if (lastCState == closed) {
      // Finished closing
      lastState = closed;
      lastPress = 0;
    } else {
      // Started Opening
      lastState = opening;
      lastPress = millis();
    }
  } else if (prevOState != lastOState) {
    if (lastOState == closed) {
      // Finished Opening
      lastState = opened;
      lastPress = 0;
    } else {
      // Started Closing
      lastState = closing;
      lastPress = millis();
    }
  }

  // Ping back to homekit server requesting a state refresh
  HTTPClient http;
  http.setTimeout(500);
  http.begin(refreshURL);
  int res = http.GET();
  if(res != HTTP_CODE_OK) {
    Serial.print("Refresh pingback failed - HTTP response code: ");
    Serial.println(res);
    Serial.println(http.getString());
  }
  http.end();
  
  return;
}

void handleRoot() {
  manageState(false, unknown);
}

void handleOpen() {
  if (!server.authenticate(httpuser, httppass)) {
    return server.requestAuthentication();
  }
  manageState(true, opened);
}

void handleClose() {
  if (!server.authenticate(httpuser, httppass)) {
    return server.requestAuthentication();
  }
  manageState(true, closed);
}

void handlePress() {
  if (!server.authenticate(httpuser, httppass)) {
    return server.requestAuthentication();
  }
  activateButton();
  server.send(200, "text/plain", "OK");
}

String stateWord(int state) {
  String s;
  switch (state) {
    case opened:
      s = "open";
      break; 
    case closed:
      s = "closed";
      break; 
    case opening:
      s = "opening";
      break; 
    case closing:
      s = "closing";
      break; 
    default:
      s = "in an unknown state";
  }
  return s;
}

void respond(int code, bool success, int status, String msg) {
  String s = "false";
  if (success) {
    s = "true";
  }
  server.send(code, "application/json", "{\"success\":" + s + ",\"status\":" + status + ",\"message\":\"" + msg + "\"}");
}

void manageState(bool change, int target) {
  int cState = digitalRead(sensorClosed);
  int oState = digitalRead(sensorOpen);
  int state = unknown;
  int code = 200;

  if (cState == closed && oState == closed) {
    respond(500, false, state, "Both door sensors report they are active!");
    return;
  }
  
  switch (cState) {
    case opened:
      if (oState == opened) {
        break;
      }
    case closed:
      state = cState;
      break;
    default:
      respond(500, false, unknown, "Unable to determine current door state");
      return;
  }
  
  if ((lastState == closing && cState == closed) || (lastState == opening && oState == closed)) {
    lastPress = 0;
  } else {
    unsigned long now = millis();
    if (now > lastPress && now < (lastPress + duration)) {
      if (change) {
        code = 400;
        change = false;
      }
      state = lastState;
    } 
  }

  switch (state) {
    case opened:
    case closed:
    case opening:
    case closing:
      break;
    default:
      respond(500, false, unknown, "Unable to determine current door state");
      return;
  }

  if (change) {
    if (target != opened && target != closed) {
      respond(400, false, state, "Unable to determine target door state from request");
      return;
    }

    if (target != state) {
      activateButton();
      lastPress = millis();
      lastState = 3 - state;
    }
  }

  respond(code, (code == 200), state, "Garage Door is currently " + stateWord(state));
}

void activateButton() {
  digitalWrite(relay, btnPush);
  delay(500);
  digitalWrite(relay, btnRelease);
}
