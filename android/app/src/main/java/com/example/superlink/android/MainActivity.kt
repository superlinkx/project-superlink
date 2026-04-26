package com.example.superlink.android

import android.content.Intent
import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import android.widget.Button
import android.widget.TextView
import android.widget.Toast

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val textView = TextView(this).apply {
            text = "Superlink Assistant Ready"
            textSize = 24f
        }

        val startButton = Button(this).apply {
            text = "Start Listening"
            setOnClickListener {
                startVoiceInteractionService(VoiceInteractionService.ACTION_START_LISTENING)
            }
        }

        val stopButton = Button(this).apply {
            text = "Stop Listening"
            setOnClickListener {
                startVoiceInteractionService(VoiceInteractionService.ACTION_STOP_LISTENING)
            }
        }

        val container = android.widget.LinearLayout(this).apply {
            orientation = android.widget.LinearLayout.VERTICAL
            addView(textView)
            addView(startButton)
            addView(stopButton)

            setPadding(32, 32, 32, 32)
        }

        setContentView(container)
    }

    private fun startVoiceInteractionService(action: String) {
        val intent = Intent(this, VoiceInteractionService::class.java).apply {
            this.action = action
        }
        startService(intent)

        val message = when (action) {
            VoiceInteractionService.ACTION_START_LISTENING -> "Listening started"
            VoiceInteractionService.ACTION_STOP_LISTENING -> "Listening stopped"
            else -> "Service action performed"
        }
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }
}
