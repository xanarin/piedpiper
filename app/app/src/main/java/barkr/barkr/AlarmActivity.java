package barkr.barkr;

import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;

public class AlarmActivity extends AppCompatActivity {

    private final String TAG = this.getClass().getSimpleName();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_alarm);
    }

    @Override
    protected void onResume(){
        super.onResume();
        Log.d(TAG, "ON RESUME");
    }

    @Override
    protected void onRestart(){
        super.onRestart();
        Log.d(TAG, "ON RESTART");
    }

    @Override
    protected void onDestroy(){
        super.onDestroy();
        Log.d(TAG, "---ON DESTROY---");
    }

    @Override
    protected void onPause(){
        super.onPause();
        Log.d(TAG, "ON PAUSE");
    }

    @Override
    protected void onStart(){
        super.onStart();
        Log.d(TAG, "ON START");
    }

    @Override
    protected void onStop(){
        super.onStop();
        Log.d(TAG, "ON STOP");
    }
}
