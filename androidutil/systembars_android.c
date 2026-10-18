#include <jni.h>
#include <stdint.h>

// 等效于设置 APPEARANCE_LIGHT_STATUS_BARS / APPEARANCE_LIGHT_NAVIGATION_BARS
void setLightSystemBars(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx) {
    JNIEnv *env = (JNIEnv *)jni_env;
    jobject activity = (jobject)ctx;

    jclass buildClass = (*env)->FindClass(env, "android/os/Build$VERSION");
    jfieldID sdkField = (*env)->GetStaticFieldID(env, buildClass, "SDK_INT", "I");
    jint sdk = (*env)->GetStaticIntField(env, buildClass, sdkField);

    jclass activityClass = (*env)->GetObjectClass(env, activity);
    jmethodID getWindowID = (*env)->GetMethodID(env, activityClass, "getWindow",
                                                "()Landroid/view/Window;");
    jobject window = (*env)->CallObjectMethod(env, activity, getWindowID);
    if (window == NULL) return;
    jclass windowClass = (*env)->GetObjectClass(env, window);

    if (sdk >= 30) {
        // Android 11+ : WindowInsetsController
        jmethodID getCtrlID = (*env)->GetMethodID(env, windowClass, "getInsetsController",
                                                   "()Landroid/view/WindowInsetsController;");
        jobject controller = (*env)->CallObjectMethod(env, window, getCtrlID);
        if (controller != NULL) {
            jclass ctrlClass = (*env)->FindClass(env, "android/view/WindowInsetsController");
            jmethodID setID = (*env)->GetMethodID(env, ctrlClass, "setSystemBarsAppearance", "(II)V");
            (*env)->CallVoidMethod(env, controller, setID, 8 | 16, 8 | 16);
            (*env)->DeleteLocalRef(env, controller);
        }
    } else if (sdk >= 23) {
        // Android 6.0~10 : decorView systemUiVisibility
        jmethodID getDecorID = (*env)->GetMethodID(env, windowClass, "getDecorView",
                                                    "()Landroid/view/View;");
        jobject decor = (*env)->CallObjectMethod(env, window, getDecorID);
        if (decor != NULL) {
            jclass viewClass = (*env)->GetObjectClass(env, decor);
            jmethodID getVisID = (*env)->GetMethodID(env, viewClass, "getSystemUiVisibility", "()I");
            jmethodID setVisID = (*env)->GetMethodID(env, viewClass, "setSystemUiVisibility", "(I)V");
            jint flags = (*env)->CallIntMethod(env, decor, getVisID);
            flags |= 0x2000;                          // SYSTEM_UI_FLAG_LIGHT_STATUS_BAR
            if (sdk >= 26) flags |= 0x10;             // SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR
            (*env)->CallVoidMethod(env, decor, setVisID, flags);
            (*env)->DeleteLocalRef(env, decor);
        }
    }
}
