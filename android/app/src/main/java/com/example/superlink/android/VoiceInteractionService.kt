package com.example.superlink.android

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.speech.RecognitionListener
import android.speech.SpeechRecognizer
import android.speech.tts.TextToSpeech
import android.speech.tts.UtteranceProgressListener
import android.util.Log
import okhttp3.*
import okhttp3.WebSocket
import org.json.JSONObject
import java.util.*

class VoiceInteractionService : Service() {
    private val TAG = "VoiceInteractionService"
    private var speechRecognizer: SpeechRecognizer? = null
    private var textToSpeech: TextToSpeech? = null
    private var webSocket: WebSocket? = null
    private var client: OkHttpClient? = null

    // Configuration constants
    companion object {
        const val ACTION_START_LISTENING = "com.example.superlink.START_LISTENING"
        const val ACTION_STOP_LISTENING = "com.example.superlink.STOP_LISTENING"
        const val ACTION_SPEAK = "com.example.superlink.SPEAK"
        const val EXTRA_TEXT = "text"

        // Default backend URL - will be overridden by environment
        private const val DEFAULT_BACKEND_URL = "ws://localhost:8080/ws"
    }

    override fun onCreate() {
        super.onCreate()
        Log.d(TAG, "VoiceInteractionService created")

        // Start as foreground service to maintain persistent connection
        startForegroundService()

        // Initialize OkHttp client for WebSocket
        client = OkHttpClient()

        // Connect to backend WebSocket
        connectWebSocket()
    }

    private fun startForegroundService() {
        val notification = android.app.Notification.Builder(this, "superlink_channel")
            .setContentTitle("Superlink Assistant")
            .setContentText("Listening for voice commands")
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .build()

        // Create notification channel
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
            val channel = android.app.NotificationChannel(
                "superlink_channel",
                "Superlink Service",
                android.app.NotificationManager.IMPORTANCE_LOW
            )
            val manager = getSystemService(android.app.NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }

        startForeground(1, notification)
    }

    private fun connectWebSocket() {
        val backendUrl = getBackendUrl()
        Log.d(TAG, "Connecting to WebSocket: $backendUrl")

        val request = Request.Builder()
            .url(backendUrl)
            .build()

        client?.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Log.d(TAG, "WebSocket connected")
                VoiceInteractionService.this.webSocket = webSocket
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                Log.d(TAG, "Received message: $text")
                handleIncomingMessage(text)
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                Log.d(TAG, "WebSocket closing: $code - $reason")
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Log.d(TAG, "WebSocket closed")
                VoiceInteractionService.this.webSocket = null
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "WebSocket error", t)
                VoiceInteractionService.this.webSocket = null
            }
        })
    }

    private fun getBackendUrl(): String {
        // Try to get from environment or use default
        val envUrl = System.getenv("BACKEND_WS_URL")
        return if (!envUrl.isNullOrEmpty()) envUrl else DEFAULT_BACKEND_URL
    }

    private fun handleIncomingMessage(message: String) {
        try {
            val json = JSONObject(message)
            if (json.has("text")) {
                val text = json.getString("text")
                speak(text)
            } else if (json.has("command") && json.getString("command") == "notify") {
                // Handle background notification from Hermes cron
                if (json.has("message")) {
                    val message = json.getString("message")
                    Log.d(TAG, "Received background notification: $message")
                    speak(message)
                }
            } else if (json.has("type") && json.getString("type") == "cron_notification") {
                // Direct cron notification from Hermes
                if (json.has("text")) {
                    val text = json.getString("text")
                    Log.d(TAG, "Received cron notification: $text")
                    speak(text)
                }
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error parsing message", e)
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START_LISTENING -> startListening()
            ACTION_STOP_LISTENING -> stopListening()
            ACTION_SPEAK -> {
                val text = intent.getStringExtra(EXTRA_TEXT)
                if (!text.isNullOrEmpty()) {
                    speak(text)
                }
            }
        }
        return START_STICKY
    }

    private fun startListening() {
        Log.d(TAG, "Starting speech recognition")
        stopListening() // Stop any existing session

        speechRecognizer = SpeechRecognizer.createSpeechRecognizer(this).apply {
            setRecognitionListener(object : RecognitionListener {
                override fun onReadyForSpeech(params: Bundle?) {
                    Log.d(TAG, "Ready for speech")
                }

                override fun onBeginningOfSpeech() {
                    Log.d(TAG, "Beginning of speech")
                }

                override fun onRmsChanged(rmsdB: Float) {
                    // Audio level changes
                }

                override fun onBufferReceived(buffer: ByteArray?) {
                    // Buffer received
                }

                override fun onEndOfSpeech() {
                    Log.d(TAG, "End of speech")
                }

                override fun onError(error: Int) {
                    val errorMessage = when (error) {
                        SpeechRecognizer.ERROR_AUDIO -> "Audio recording error"
                        SpeechRecognizer.ERROR_CLIENT -> "Client side error"
                        SpeechRecognizer.ERROR_INSUFFICIENT_PERMISSIONS -> "Insufficient permissions"
                        SpeechRecognizer.ERROR_NETWORK -> "Network error"
                        SpeechRecognizer.ERROR_NETWORK_TIMEOUT -> "Network timeout"
                        SpeechRecognizer.ERROR_NO_MATCH -> "No recognition match"
                        SpeechRecognizer.ERROR_RECOGNIZER_BUSY -> "RecognitionService busy"
                        SpeechRecognizer.ERROR_SERVER -> "Server error"
                        SpeechRecognizer.ERROR_SPEECH_TIMEOUT -> "No speech input"
                        else -> "Unknown error: $error"
                    }
                    Log.e(TAG, "Speech recognition error: $errorMessage")
                }

                override fun onResults(results: Bundle?) {
                    val matches = results?.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION)
                    if (!matches.isNullOrEmpty()) {
                        val spokenText = matches[0]
                        Log.d(TAG, "Recognized text: $spokenText")

                        // Send to backend via WebSocket
                        sendToBackend(spokenText)

                        // Also speak it back for confirmation (can be removed in production)
                        speak("You said: $spokenText")
                    }
                }

                override fun onPartialResults(partialResults: Bundle?) {
                    // Partial recognition results
                }

                override fun onEvent(eventType: Int, params: Bundle?) {
                    // Event handling
                }
            })

            val intent = Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH).apply {
                putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM)
                putExtra(RecognizerIntent.EXTRA_CALLING_PACKAGE, packageName)
                putExtra(RecognizerIntent.EXTRA_MAX_RESULTS, 1)
            }
            startListening(intent)
        }
    }

    private fun sendToBackend(text: String) {
        val message = JSONObject().apply {
            put("type", "user_message")
            put("text", text)
            put("timestamp", System.currentTimeMillis())
        }.toString()

        webSocket?.send(message)
        Log.d(TAG, "Sent to backend: $message")
    }

    private fun speak(text: String) {
        if (textToSpeech == null) {
            textToSpeech = TextToSpeech(this) { status ->
                if (status == TextToSpeech.SUCCESS) {
                    Log.d(TAG, "TTS initialized successfully")
                } else {
                    Log.e(TAG, "TTS initialization failed")
                }
            }
        }

        // Speak the text
        val utteranceId = UUID.randomUUID().toString()
        textToSpeech?.speak(text, TextToSpeech.QUEUE_FLUSH, null, utteranceId)
        Log.d(TAG, "Speaking: $text")
    }

    private fun stopListening() {
        speechRecognizer?.let { recognizer ->
            if (recognizer.isRecognitionAvailable) {
                recognizer.stopListening()
                recognizer.destroy()
            }
        }
        speechRecognizer = null
        Log.d(TAG, "Stopped speech recognition")
    }

    override fun onDestroy() {
        stopListening()
        textToSpeech?.shutdown()
        webSocket?.close(1000, "Service destroyed")
        client?.dispatcher?.executorService?.shutdown()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }
}