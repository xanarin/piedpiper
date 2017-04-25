package netsec.PiedPiper;

import java.io.ByteArrayOutputStream;
import java.security.SecureRandom;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.SecretKeySpec;

/**
 * Created by yupyupp on 4/24/17.
 */

public class SimpleCrypto {

    public static byte[] generateKey(String password) {
        KeyGenerator kgen;
        SecureRandom sr;

        // Get bytes of password string
        byte[] keyStart = password.getBytes();

        //Initalize the key geneartor and the random number generator
        try {
            kgen = KeyGenerator.getInstance("AES/CBC/PKCS5Padding");
            sr = SecureRandom.getInstance("PBKDF2WithHmacSHA1");
        } catch (Exception e) {
            e.printStackTrace();
            return null;
        }

        // Seed the RNG with the password and request a 256 bit AES key
        sr.setSeed(keyStart);
        kgen.init(256, sr);
        SecretKey skey = kgen.generateKey();

        // Return bytearray of encoded key
        return skey.getEncoded();
    }

    public static byte[] encrypt(byte[] aesKey, byte[] plainText) throws Exception {
        SecretKeySpec skeySpec = new SecretKeySpec(aesKey, "AES/CBC/PKCS5Padding");
        Cipher cipher = Cipher.getInstance("AES/CBC/PKCS5Padding");
        cipher.init(Cipher.ENCRYPT_MODE, skeySpec);
        return cipher.doFinal(plainText);
    }

    public static byte[] decrypt(byte[] aesKey, byte[] cipherText) throws Exception {
        SecretKeySpec skeySpec = new SecretKeySpec(aesKey, "AES/CBC/PKCS5Padding");
        Cipher cipher = Cipher.getInstance("AES/CBC/PKCS5Padding");
        cipher.init(Cipher.DECRYPT_MODE, skeySpec);
        return cipher.doFinal(cipherText);
    }
}
