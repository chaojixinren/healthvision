package com.healthvision.app;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.os.Build;
import android.os.Bundle;
import android.webkit.WebView;
import com.getcapacitor.BridgeActivity;
import com.capacitorjs.plugins.geolocation.GeolocationPlugin;

public class MainActivity extends BridgeActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        registerPlugin(GeolocationPlugin.class);
        super.onCreate(savedInstanceState);
        if ((getApplicationInfo().flags & android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0) {
            WebView.setWebContentsDebuggingEnabled(true);
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            NotificationChannel remindersChannel = new NotificationChannel(
                "reminders",
                "服药提醒",
                NotificationManager.IMPORTANCE_HIGH
            );
            remindersChannel.setDescription("按时服药的通知提醒");

            NotificationChannel proximityChannel = new NotificationChannel(
                "device-proximity",
                "药箱距离提醒",
                NotificationManager.IMPORTANCE_HIGH
            );
            proximityChannel.setDescription("药箱距离过远或设备离线时提醒");

            NotificationManager manager = getSystemService(NotificationManager.class);
            manager.createNotificationChannel(remindersChannel);
            manager.createNotificationChannel(proximityChannel);
        }
    }
}
