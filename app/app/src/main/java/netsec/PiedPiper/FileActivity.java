package netsec.PiedPiper;

import android.content.DialogInterface;
import android.content.SharedPreferences;
import android.os.AsyncTask;
import android.os.Bundle;
import android.os.Environment;
import android.support.v7.app.AlertDialog;
import android.support.v7.app.AppCompatActivity;
import android.text.InputType;
import android.util.Log;
import android.view.View;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.EditText;
import android.widget.Spinner;
import android.widget.TextView;

import org.apache.http.HttpResponse;
import org.apache.http.client.HttpClient;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.entity.ByteArrayEntity;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.client.DefaultHttpClient;
import org.apache.http.util.EntityUtils;
import org.json.JSONObject;

import java.io.BufferedInputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashSet;
import java.util.Set;
import java.util.TimeZone;

public class FileActivity extends AppCompatActivity {

    private Button mChooseButton;
    private Button mUploadButton;
    private Button mDownloadButton;
    private Spinner _downloadSipnner;
    private TextView fileTxt;

    private static final String SHARED_PREF_FILE = "PiedPiperSettings";
    private SharedPreferences sharedPreferences;
    private String _token;
    private String _username;
    private String _fileName;
    private String _fileNameDown;
    private String _fullPath;
    Set<String> _uploadedFiles;


    private byte[] _aesKey;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_file);


        // Gest password for AES keys building
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("Enter file encryption passphrase");

        // Set up the input
        final EditText input = new EditText(this);
        // Specify the type of input expected; this, for example, sets the input as a password, and will mask the text
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        builder.setView(input);

        // Set up the buttons
        builder.setPositiveButton("OK", new DialogInterface.OnClickListener() {
            @Override
            public void onClick(DialogInterface dialog, int which) {
                _aesKey = SimpleCrypto.generateKey(input.getText().toString());
            }
        });
        builder.setNegativeButton("Cancel", new DialogInterface.OnClickListener() {
            @Override
            public void onClick(DialogInterface dialog, int which) {
                dialog.cancel();
            }
        });
        builder.show();


        sharedPreferences=getSharedPreferences(SHARED_PREF_FILE,0);
        _token = sharedPreferences.getString("token","NO_TOKEN");
        _username = sharedPreferences.getString("username","NO_USERNAME");
        _uploadedFiles = sharedPreferences.getStringSet("uploaded",new HashSet<String>());

        _downloadSipnner = (Spinner) findViewById(R.id.downloadSelect);
        ArrayList<String> downloadList = new ArrayList<String>();
        downloadList.addAll(_uploadedFiles);
        ArrayAdapter<String> adapter = new ArrayAdapter<String>(this, android.R.layout.simple_spinner_item, downloadList);
        _downloadSipnner.setAdapter(adapter);

        fileTxt = (TextView) findViewById(R.id.textFileUp);

        mChooseButton = (Button)findViewById(R.id.buttonFileChoose);
        mChooseButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                new FileChooser(FileActivity.this).setFileListener(new FileChooser.FileSelectedListener() {
                    @Override public void fileSelected(final File file) {
                        // do something with the file
                        Log.i("FileChooser", file.getName());
                        _fileName = file.getName();
                        _fullPath = file.getAbsolutePath();
                        fileTxt.setText(file.getAbsolutePath());
                    }}).showDialog();
            }
        });

        mUploadButton = (Button)findViewById(R.id.buttonFileUpload);
        mUploadButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                UploadAsync uploadTask = new UploadAsync();
                uploadTask.execute();
            }
        });

        mDownloadButton = (Button)findViewById(R.id.buttonFileDownload);
        mDownloadButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                DownloadAsync downloadTask = new DownloadAsync();
                downloadTask.execute();
            }
        });
    }
    class UploadAsync extends AsyncTask<Void, Void, Void> {

        @Override
        protected Void doInBackground(Void... params) {

            Log.e("Entering doInBackground", "");

            //encrypt
            byte[] cipherText = {};
            try {
                File fi = new  File(_fullPath);
                cipherText = SimpleCrypto.encrypt(_aesKey, read(fi));
                Log.i("Encrypted","wo");
            }
            catch (Exception e) {
                Log.e("Upload", "read from file or encrypt", e);
            }

            //create object
            String file_id = createObject(_token, _fileName);

            //upload
            uploadObject(file_id, cipherText);
            return null;
        }

        @Override
        protected void onPostExecute(Void aVoid) {
            super.onPostExecute(aVoid);
            _uploadedFiles.add(_fileName);
            sharedPreferences=getSharedPreferences(SHARED_PREF_FILE,0);
            SharedPreferences.Editor editor=sharedPreferences.edit();
            editor.putStringSet("uploaded",_uploadedFiles);
            editor.commit();

            Log.i("UploadAsync","fin");
        }
    }



    class DownloadAsync extends AsyncTask<Void, Void, Void> {

        @Override
        protected Void doInBackground(Void... params) {

            Log.e("Entering *download* bg", "");

            try {
                byte[] cipherText = getObject(_token, _fileNameDown);
                Log.i("Downed","wo");
                byte[] plainText = SimpleCrypto.decrypt(_aesKey, cipherText);
                Log.i("Decrypted","wo");
                saveFile(plainText, _fileNameDown);
                Log.i("Saved","wo");
            }
            catch (Exception e) {
                Log.e("Download", "write to file or decrypt", e);
            }
            return null;
        }

        @Override
        protected void onPostExecute(Void aVoid) {
            super.onPostExecute(aVoid);
            // TODO
            Log.i("DownloadAsync","fin");
        }
    }

    private String createObject(String token, String filename) {
        String responseServer= "";
        HttpURLConnection urlConnection=null;
        String json = null;
        String reply = null;
        try {
            SimpleDateFormat dateFormatGmt = new SimpleDateFormat("yyyyMMddHHmmss");
            dateFormatGmt.setTimeZone(TimeZone.getTimeZone("GMT"));
            Date now = new Date();

            HttpResponse response;
            JSONObject jsonObject = new JSONObject();
            jsonObject.accumulate("token", token);
            jsonObject.accumulate("filename", filename);
            json = jsonObject.toString();
            Log.i("Posting:", json);
            HttpClient httpClient = new DefaultHttpClient();
            HttpPost httpPost = new HttpPost("https://pp.848.productions/object");
            httpPost.setEntity(new StringEntity(json, "UTF-8"));
            httpPost.setHeader("Content-Type", "application/json");
            httpPost.setHeader("Accept-Encoding", "application/json");
            httpPost.setHeader("Accept-Language", "en-US");
            response = httpClient.execute(httpPost);
            Log.i("response", response.getStatusLine().getReasonPhrase());

            InputStream inputStream = response.getEntity().getContent();
            MenuActivity.StringifyStream str = new MenuActivity.StringifyStream();
            responseServer = str.getStringFromInputStream(inputStream);
            Log.d("GetToken Server Reply", responseServer);
            JSONObject replyJson = new JSONObject(responseServer);

            Log.e("response", responseServer);

        } catch (Exception e) {
            e.printStackTrace();
        }
        return responseServer;
    }

    private int uploadObject(String objectID, byte[] cipherText) {
        String responseServer;

        HttpURLConnection urlConnection=null;
        String json = null;
        String reply = null;
        int responseCode = 8888;

        try {
            HttpResponse response;
            HttpClient httpClient = new DefaultHttpClient();
            HttpPost httpPost = new HttpPost("https://pp.848.productions/object/" + objectID);
            httpPost.setEntity(new ByteArrayEntity(cipherText));
            httpPost.setHeader("Content-Type", "application/json");
            httpPost.setHeader("Accept-Encoding", "application/json");
            httpPost.setHeader("Accept-Language", "en-US");
            response = httpClient.execute(httpPost);

            responseCode = response.getStatusLine().getStatusCode();

            Log.i("response", response.getStatusLine().getReasonPhrase());

            InputStream inputStream = response.getEntity().getContent();
            MenuActivity.StringifyStream str = new MenuActivity.StringifyStream();
            responseServer = str.getStringFromInputStream(inputStream);
            Log.d("GetToken Server Reply", responseServer);

            Log.e("response", responseServer);
            //cipherText = "".getBytes();

        } catch (Exception e) {
            e.printStackTrace();
        }
        return responseCode;
    }

    private byte[] getObject(String token, String filename) {
        String responseServer = "";

        HttpURLConnection urlConnection=null;
        String json = null;
        String reply = null;
        byte[] cipherText = null;
        try {
            SimpleDateFormat dateFormatGmt = new SimpleDateFormat("yyyyMMddHHmmss");
            dateFormatGmt.setTimeZone(TimeZone.getTimeZone("GMT"));
            Date now = new Date();

            HttpResponse response;
            JSONObject jsonObject = new JSONObject();
            jsonObject.accumulate("token", token);
            jsonObject.accumulate("filename", filename);
            json = jsonObject.toString();
            Log.i("getting:", json);

            HttpClient httpClient = new DefaultHttpClient();

            HttpGetWithEntity  httpGet = new HttpGetWithEntity ("https://pp.848.productions/object");
            httpGet.setEntity(new StringEntity(json, "UTF-8"));

            response = httpClient.execute(httpGet);
            Log.i("response", response.getStatusLine().getReasonPhrase());
            cipherText = EntityUtils.toByteArray(response.getEntity());
//                InputStream inputStream = response.getEntity().getContent();
//                StringifyStream str = new StringifyStream();
//                responseServer = str.getStringFromInputStream(inputStream);
            responseServer = cipherText.toString();
            Log.d("GetToken Server Reply", responseServer);
            //JSONObject replyJson = new JSONObject(responseServer);
            //token = getHashCodeFromString(username + replyJson.getString("Nonce") + jsonObject.getString("foo"));
//                cipherText = responseServer.getBytes();
            Log.e("response", responseServer);

        } catch (Exception e) {
            e.printStackTrace();
        }
        return cipherText;
    }

    private String convertFile() {

        String responseServer = "";

        final File file = new File(Environment.getExternalStorageDirectory().getAbsolutePath(), "file.txt");
        int size = (int) file.length();
        byte[] bytes = new byte[size];
        try {
            BufferedInputStream buf = new BufferedInputStream(new FileInputStream(file));
            buf.read(bytes, 0, bytes.length);
            buf.close();
        } catch (Exception e) {
            e.printStackTrace();
        }
        //TODO VVV
        //plainText = bytes;
        String rtn = new String();
        return responseServer;
    }

    private String saveFile(byte[] plainText, String filename) {

        File file=new File(Environment.getExternalStorageDirectory(), filename);
        try {
            file.createNewFile();
        } catch (java.io.IOException e) {
            Log.e("SaveFile", file.getAbsolutePath(), e);
        }

        try {
            FileOutputStream fos=new FileOutputStream(file.getPath());
            fos.write(plainText);
            fos.close();
        }
        catch (java.io.IOException e) {
            Log.e("saveFile", "Write to file", e);
        }

        return "saved";
    }

    public byte[] read(File file) {
        byte[] buffer = new byte[(int) file.length()];
        InputStream ios = null;
        try {
            ios = new FileInputStream(file);
            if (ios.read(buffer) == -1) {
                throw new IOException("EOF reached while trying to read the whole file");
            }
        } catch (Exception e) {
            Log.e("Read","Error readin into bytearray", e);
        } finally {
            try {
                if (ios != null)
                    ios.close();
            } catch (IOException e) {
            }
        }
        return buffer;
    }
}
