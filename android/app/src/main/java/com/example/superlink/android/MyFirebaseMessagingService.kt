package com.example.superlink.android

import android.util.Log
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

class MyFirebaseMessagingService : FirebaseMessagingService() {
    private val TAG = "MyFirebaseMsgService"

    override fun onNewToken(token: String) {
        Log.d(TAG, "Refreshed token: $token")
        // Send this token to your backend if needed
    }

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        Log.d(TAG, "From: ${remoteMessage.from}")

        // Check if message contains a data payload.
        remoteMessage.data?.let { data ->
            Log.d(TAG, "Message data payload: $data")

            // Handle background notification from Hermes
            val text = data["text"]
            val command = data["command"]

            if (command == "notify" && !text.isNullOrEmpty()) {
                Log.d(TAG, "Received background notification: $text")
                handleBackgroundNotification(text)
            }
        }

        // Check if message contains a notification payload.
        remoteMessage.notification?.let { notification ->
            Log.d(TAG, "Message notification body: ${notification.body}")
        }
    }

    private fun handleBackgroundNotification(message: String) {
        // Start the VoiceInteractionService to speak the notification
        val intent = android.content.Intent(this, VoiceInteractionService::class.java).apply {
            action = VoiceInteractionService.ACTION_SPEAK
            putExtra(VoiceInteractionService.EXTRA_TEXT, message)
        }
        startService(intent)
    }
}