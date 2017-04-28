package netsec.PiedPiper;

import android.content.Intent;
import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.widget.Button;
import android.widget.TextView;

import java.io.File;

public class FileActivity extends AppCompatActivity {

    private Button mChooseButton;
    private Button mUploadButton;
    private TextView fileTxt;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_file);

        fileTxt = (TextView) findViewById(R.id.textFileUp);


        mChooseButton = (Button)findViewById(R.id.buttonFileChoose);
        mChooseButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                new FileChooser(FileActivity.this).setFileListener(new FileChooser.FileSelectedListener() {
                    @Override public void fileSelected(final File file) {
                        // do something with the file
                        Log.i("FileChooser", file.getName());
                        fileTxt.setText(file.getAbsolutePath());
                    }}).showDialog();
            }
        });


    }
}
