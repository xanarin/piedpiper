package netsec.PiedPiper;

import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.widget.Button;

import org.apache.http.client.methods.HttpGet;
import org.json.JSONObject;
import android.os.AsyncTask;
import android.widget.TextView;
import org.apache.http.HttpResponse;
import org.apache.http.client.HttpClient;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.impl.client.DefaultHttpClient;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.text.SimpleDateFormat;
import java.util.Date;

import java.net.HttpURLConnection;
import java.util.TimeZone;

import org.apache.http.entity.StringEntity;


public class MenuActivity extends AppCompatActivity {

    enum ServerAction {
        USER_REGISTER,
        REQUEST_TOKEN
    }

    private final String TAG = this.getClass().getSimpleName();

    private Button mRegisterButton;
    private Button mTokenButton;
    private Button mEncryptButton;
    private Button mDecryptButton;

    private byte[] plainText;
    private byte[] cipherText;

    private byte[] aesKey;

    String responseServer;
    TextView txt;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_menu);

        txt = (TextView) findViewById(R.id.text);

        aesKey = SimpleCrypto.generateKey("Thisismypassword");
        if (aesKey == null) {
            Log.e("onCreate", "Unable to generate key");
        }
        plainText = "This is my plaintext".getBytes();
        cipherText = "".getBytes();

        mRegisterButton = (Button)findViewById(R.id.userRegister);
        mRegisterButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.USER_REGISTER);
            }
        });

        mTokenButton = (Button)findViewById(R.id.getToken);
        mTokenButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                ProcessButton processButton = new ProcessButton();
                processButton.execute(ServerAction.REQUEST_TOKEN);
            }
        });

        mEncryptButton = (Button)findViewById(R.id.encrypt);
        mEncryptButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                try {
                    final byte[] finalPlain = plainText.clone();
                    cipherText = SimpleCrypto.encrypt(aesKey, finalPlain);
                    Log.i("Encrypt - Plain", SimpleCrypto.bytesToHex(plainText));
                    Log.i("Encrypt - Cipher", SimpleCrypto.bytesToHex(cipherText));

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
                } catch (Exception e) {
                    Log.e("Decrypt", e.toString());
                }
            }
        });

    }
    /* Inner class to get response */
    class ProcessButton extends AsyncTask<ServerAction, Void, Void> {

        private String userRegister(String user, String pass) {
            HttpURLConnection urlConnection=null;
            String json = null;
            String reply = null;
            try {
                //Create User

                HttpResponse response;
                JSONObject jsonObject = new JSONObject();
                jsonObject.accumulate("username", user);
                jsonObject.accumulate("password", pass);
                json = jsonObject.toString();
                HttpClient httpClient = new DefaultHttpClient();
                HttpPost httpPost = new HttpPost("https://pp.848.productions/user");
                httpPost.setEntity(new StringEntity(json, "UTF-8"));
                httpPost.setHeader("Content-Type", "application/json");
                httpPost.setHeader("Accept-Encoding", "application/json");
                httpPost.setHeader("Accept-Language", "en-US");
                response = httpClient.execute(httpPost);
                InputStream inputStream = response.getEntity().getContent();
                StringifyStream str = new StringifyStream();
                reply = str.getStringFromInputStream(inputStream);
            } catch (Exception e) {
                e.printStackTrace();
            }
            return reply;
        }

        private String requestToken(String username, String password) {
            HttpURLConnection urlConnection=null;
            String json = null;
            String reply = null;
            try {
                SimpleDateFormat dateFormatGmt = new SimpleDateFormat("yyyyMMddHHmmss");
                dateFormatGmt.setTimeZone(TimeZone.getTimeZone("GMT"));
                Date now = new Date();

                HttpResponse response;
                JSONObject jsonObject = new JSONObject();
                jsonObject.accumulate("reqdate", dateFormatGmt.format(now));
                jsonObject.accumulate("username", username);
                jsonObject.accumulate("password", password);
                json = jsonObject.toString();
                Log.i("getting:", json);
                HttpClient httpClient = new DefaultHttpClient();
                HttpGet httpGet = new HttpGet("https://pp.848.productions/auth");
                //httpGet.setEntity(new StringEntity(json, "UTF-8"));
                httpGet.setHeader("Content-Type", "application/json");
                httpGet.setHeader("Accept-Encoding", "application/json");
                httpGet.setHeader("Accept-Language", "en-US");
                response = httpClient.execute(httpGet);
                Log.i("response", response.getStatusLine().getReasonPhrase());

                InputStream inputStream = response.getEntity().getContent();
                StringifyStream str = new StringifyStream();
                responseServer = str.getStringFromInputStream(inputStream);
                Log.d("GetToken Server Reply", responseServer);
                JSONObject replyJson = new JSONObject(responseServer);
                reply = getHashCodeFromString(username + replyJson.getString("nonce") + jsonObject.getString("reqdate"));

                Log.e("response", responseServer);

            } catch (Exception e) {
                e.printStackTrace();
            }
            return reply;
        }

        @Override
        protected Void doInBackground(ServerAction... params) {

            Log.e("Entering doInBackground", params[0].name());

            HttpURLConnection urlConnection=null;
            String json = null;
            ServerAction action = params[0];
            String username = "user1234";
            String password = "pass1234";

            switch (action) {
                case USER_REGISTER:
                    responseServer = userRegister(username, password);
                    break;
                case REQUEST_TOKEN:
                    responseServer = requestToken(username, password);
                    break;
                default:
                    responseServer = "Action not registered";
            }

            return null;
        }

        @Override
        protected void onPostExecute(Void aVoid) {
            super.onPostExecute(aVoid);

            txt.setText(responseServer);
        }
    }

    public static class StringifyStream {

        public static void main(String[] args) throws IOException {
            InputStream is = new ByteArrayInputStream("".getBytes());

            String result = getStringFromInputStream(is);

            System.out.println(result);
            System.out.println("Done");

        }

        // convert InputStream to String
        private static String getStringFromInputStream(InputStream is) {

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

    private static String getHashCodeFromString(String str) throws NoSuchAlgorithmException {
        MessageDigest md = MessageDigest.getInstance("SHA-512");
        md.update(str.getBytes());
        byte byteData[] = md.digest();

        //convert the byte to hex format method 1
        StringBuffer hashCodeBuffer = new StringBuffer();
        for (int i = 0; i < byteData.length; i++) {
            hashCodeBuffer.append(Integer.toString((byteData[i] & 0xff) + 0x100, 16).substring(1));
        }
        return hashCodeBuffer.toString();
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
