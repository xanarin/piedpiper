package netsec.PiedPiper;

import android.util.Log;

import java.io.ByteArrayOutputStream;
import java.security.Key;
import java.security.SecureRandom;
import java.security.spec.KeySpec;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.SecretKeyFactory;
import javax.crypto.spec.PBEKeySpec;
import javax.crypto.spec.PBEParameterSpec;
import javax.crypto.spec.SecretKeySpec;

import static android.R.id.input;

/**
 * Created by yupyupp on 4/24/17.
 */

public class SimpleCrypto {

    // TODO - I know this is very bad crypto (SHA1PRNG)
    public static byte[] generateKey(String password) {
        byte[] salt = {
                (byte)0xc7, (byte)0x73, (byte)0x21, (byte)0x8c,
                (byte)0x7e, (byte)0xc8, (byte)0xee, (byte)0x99
        };
        /*
        try {




            SecretKeyFactory factory = SecretKeyFactory.getInstance("AES");
            KeySpec spec = new PBEKeySpec(password.toCharArray(), salt, 65536, 256);
            SecretKey tmp = factory.generateSecret(spec);
            SecretKey secret = new SecretKeySpec(tmp.getEncoded(), "AES");

            KeyGenerator kgen;
            SecureRandom sr;

            pbeKeySpec = new PBEKeySpec(password.toCharArray());
            keyFac = SecretKeyFactory.getInstance("AES");
            SecretKey pbeKey = keyFac.generateSecret(pbeKeySpec);

            Log.d("KeyGen", bytesToHex(secret.getEncoded()));
            return secret.getEncoded();

        } catch (Exception e) {
            Log.e("generateKey", "Error generating key");
            e.printStackTrace();
            return null;
        }
        */
        // Get bytes of password string
        byte[] keyStart = password.getBytes();
        KeyGenerator kgen = null;
        SecureRandom sr = null;

        //Initalize the key geneartor and the random number generator
        try {
            SecretKeyFactory f = SecretKeyFactory.getInstance("PBKDF2WithHmacSHA1");
            KeySpec ks = new PBEKeySpec(password.toCharArray(),salt,1024,256);
            SecretKey s = f.generateSecret(ks);
            Key k = new SecretKeySpec(s.getEncoded(),"AES");
            Log.d("key", bytesToHex(k.getEncoded()));
            return k.getEncoded();

            /*
            kgen = KeyGenerator.getInstance("AES");
            sr = SecureRandom.getInstance("SHA1PRNG");
            */
        } catch (Exception e) {
            Log.e("generateKey", "Error generating key");
            e.printStackTrace();
            return null;
        }
        /*
        // Seed the RNG with the password and request a 256 bit AES key
        sr.setSeed(keyStart);
        kgen.init(256, sr);
        SecretKey skey = kgen.generateKey();
        Log.d("KeyGen", bytesToHex(skey.getEncoded()));

        // Return bytearray of encoded key
        return skey.getEncoded();
        */
    }

    // TODO - I know this is very bad crypto (AES/ECb)
    public static byte[] encrypt(byte[] aesKey, byte[] plainText) throws Exception {
        SecretKeySpec skeySpec = new SecretKeySpec(aesKey, "AES");
        Cipher cipher = Cipher.getInstance("AES");
        cipher.init(Cipher.ENCRYPT_MODE, skeySpec);
        byte[] cipherText = cipher.doFinal(plainText);
        Log.i("EncryptLen", "" + cipherText.length);
        return cipherText;
    }

    // TODO - I know this is very bad crypto (AES/ECb)
    public static byte[] decrypt(byte[] aesKey, byte[] cipherText) throws Exception {
        Log.i("DecryptLen", "" + cipherText.length);
        SecretKeySpec skeySpec = new SecretKeySpec(aesKey, "AES");
        Cipher cipher = Cipher.getInstance("AES");
        cipher.init(Cipher.DECRYPT_MODE, skeySpec);
        return cipher.doFinal(cipherText);
    }

    private final static char[] hexArray = "0123456789ABCDEF".toCharArray();
    public static String bytesToHex(byte[] bytes) {
        char[] hexChars = new char[bytes.length * 2];
        for ( int j = 0; j < bytes.length; j++ ) {
            int v = bytes[j] & 0xFF;
            hexChars[j * 2] = hexArray[v >>> 4];
            hexChars[j * 2 + 1] = hexArray[v & 0x0F];
        }
        return new String(hexChars);
    }
}
