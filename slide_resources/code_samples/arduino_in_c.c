
void loop() {

  // put your main code here, to run repeatedly:
  unsigned long currentTime = millis();

  // task 1 // HL
  if(currentTime - previousTimeLed1 > timeIntervalLed1) {
    previousTimeLed1 = currentTime;
    if (ledState1 == HIGH) {
      ledState1 = LOW; // HL
    }
    else {
      ledState1 = HIGH; // HL
    }
  }
  // task 2 // HL
  if (Serial.available()) {
    int userInput = Serial.parseInt();


