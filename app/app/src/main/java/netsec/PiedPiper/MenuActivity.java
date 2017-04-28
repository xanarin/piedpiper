package netsec.PiedPiper;

import android.Manifest;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Environment;
import android.support.annotation.RequiresApi;
import android.support.v4.app.ActivityCompat;
import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.widget.Button;

import org.apache.http.entity.ByteArrayEntity;
import org.json.JSONObject;
import android.os.AsyncTask;
import android.widget.TextView;

import org.apache.http.HttpResponse;
import org.apache.http.client.HttpClient;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.impl.client.DefaultHttpClient;

import java.io.BufferedInputStream;
import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.text.SimpleDateFormat;
import java.util.Date;

import java.net.HttpURLConnection;
import java.util.TimeZone;

import org.apache.http.entity.StringEntity;
import org.apache.http.util.EntityUtils;



public class MenuActivity extends AppCompatActivity {

    enum ServerAction {
        USER_REGISTER,
        REQUEST_TOKEN,
        CREATE_OBJECT,
        UPLOAD_OBJECT,
        GET_OBJECT,
        CONVERT_FILE,
        SAVE_FILE
    }

    private final String TAG = this.getClass().getSimpleName();

    private Button mUserButton;
    private Button mFileButton;

    private Button mEncryptButton;
    private Button mDecryptButton;
    private Button mCreateObject;
    private Button mUploadObject;
    private Button mGetObject;
    private Button mConvertFile;
    private Button mSaveFile;


    private byte[] plainText;
    private byte[] cipherText;
    private byte[] test = "Sample Test File".getBytes();

    private byte[] aesKey;

    String responseServer;
    String objectID;
    TextView txt;

    @RequiresApi(api = Build.VERSION_CODES.M)
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        boolean needsRead = ActivityCompat.checkSelfPermission(this, Manifest.permission.READ_EXTERNAL_STORAGE)
                != PackageManager.PERMISSION_GRANTED;
        if (needsRead) {
            requestPermissions(new String[]{Manifest.permission.READ_EXTERNAL_STORAGE}, 1);
        }

        boolean needsWrite = ActivityCompat.checkSelfPermission(this, Manifest.permission.WRITE_EXTERNAL_STORAGE)
                != PackageManager.PERMISSION_GRANTED;
        if (needsWrite) {
            requestPermissions(new String[]{Manifest.permission.WRITE_EXTERNAL_STORAGE}, 1);
        }


        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_menu);

        txt = (TextView) findViewById(R.id.text);

        mUserButton = (Button)findViewById(R.id.user);
        mUserButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                Intent toLogin = new Intent(MenuActivity.this, LoginActivity.class);
                startActivity(toLogin);
            }
        });


        mFileButton = (Button)findViewById(R.id.file);
        mFileButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                Intent toFile = new Intent(MenuActivity.this, FileActivity.class);
                startActivity(toFile);
            }
        });


        /*
        mEncryptButton = (Button)findViewById(R.id.encrypt);
        mEncryptButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                try {
                    final byte[] finalPlain = plainText.clone();
                    cipherText = SimpleCrypto.encrypt(aesKey, finalPlain);
                    plainText = "stop".getBytes();
                    Log.i("Encrypt - Plain", SimpleCrypto.bytesToHex(plainText));
                    Log.i("Encrypt - Cipher", SimpleCrypto.bytesToHex(cipherText));
                    txt.setText("Plain: " + SimpleCrypto.bytesToHex(plainText) + "\nCipher: " + SimpleCrypto.bytesToHex(cipherText));

                } catch (Exception e) {
                    Log.e("Encrypt", e.toString());
                }
            }
        });
        mDecryptButton = (Button)findViewById(R.id.decrypt);
        mDecryptButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                try {
                    final byte[] finalCipher = cipherText.clone();
                    plainText = SimpleCrypto.decrypt(aesKey, finalCipher);
                    Log.i("Decrypt - Cipher", SimpleCrypto.bytesToHex(cipherText));
                    Log.i("Decrypt - Plain", SimpleCrypto.bytesToHex(plainText));
                    txt.setText("Cipher: " + SimpleCrypto.bytesToHex(cipherText) + "\nPlain: " + SimpleCrypto.bytesToHex(plainText));

                } catch (Exception e) {
                    Log.e("Decrypt", e.toString());
                }
            }
        });
        mCreateObject = (Button)findViewById(R.id.createObject);
        mCreateObject.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.CREATE_OBJECT);
            }
        });
        mUploadObject = (Button)findViewById(R.id.uploadObject);
        mUploadObject.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.UPLOAD_OBJECT);
            }
        });
        mGetObject = (Button)findViewById(R.id.getObject);
        mGetObject.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.GET_OBJECT);
            }
        });

        mConvertFile = (Button)findViewById(R.id.convertFile);
        mConvertFile.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.CONVERT_FILE);
            }
        });
        mSaveFile = (Button)findViewById(R.id.saveFile);
        mSaveFile.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.SAVE_FILE);
            }
        });
        */

    }

    public static class StringifyStream {

        public static void main(String[] args) throws IOException {
            InputStream is = new ByteArrayInputStream("".getBytes());

            String result = getStringFromInputStream(is);

            System.out.println(result);
            System.out.println("Done");

        }

        // convert InputStream to String
        public static String getStringFromInputStream(InputStream is) {

            BufferedReader b_reader = null;
            StringBuilder s_builder = new StringBuilder();

            String line;
            try {
                b_reader = new BufferedReader(new InputStreamReader(is));
                while ((line = b_reader.readLine()) != null) {
                    s_builder.append(line);
                }
            } catch (IOException e) {
                e.printStackTrace();
            } finally {
                if (b_reader != null) {
                    try {
                        b_reader.close();
                    } catch (IOException e) {
                        e.printStackTrace();
                    }
                }
            }
            return s_builder.toString();
        }

    }

    /* Inner class to get response */


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
