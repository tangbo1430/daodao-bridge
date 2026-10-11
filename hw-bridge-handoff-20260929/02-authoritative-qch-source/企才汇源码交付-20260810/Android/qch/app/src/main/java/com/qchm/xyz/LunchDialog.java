package com.qchm.xyz;

import android.app.Dialog;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.text.SpannableString;
import android.text.Spanned;
import android.text.TextPaint;
import android.text.method.LinkMovementMethod;
import android.text.style.ClickableSpan;
import android.view.Gravity;
import android.view.View;
import android.view.WindowManager;
import android.widget.TextView;

import androidx.annotation.NonNull;

import com.qchm.xyz.bj.view.X5WebViewActivity;

public class LunchDialog extends Dialog {
    Context context;
    View.OnClickListener left, right;
    TextView  tvTips, leftText, rightText;
    String tips = "感谢您选择企才汇！\n我们非常重视您的个人信息和隐私保护。使用产品前，请认真阅读并充分理解《用户协议》和《隐私政策》。";
    String clickText1 = "《用户协议》";
    String clickText2 = "《隐私政策》";
 
    public LunchDialog(@NonNull Context context, View.OnClickListener left, View.OnClickListener right) {
        super(context, R.style.myChooseDialog);
        this.context = context;
        this.left = right;
        this.right = left;
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.dialog_lunch);
        setWidows();
        tvTips = findViewById(R.id.tvTips);
        SpannableString spannableString = new SpannableString(tips);
        int color = context.getResources().getColor(R.color.qch_purple);
        int start = 0;
        int end = 0;
        start = tips.indexOf(clickText1);
        end = start + clickText1.length();
        spannableString.setSpan(new ClickableSpan() {
            @Override
            public void onClick(@NonNull View widget) {
                Intent intent = new Intent(context, X5WebViewActivity.class);
                intent.putExtra("title", "用户协议");
                intent.putExtra("url", DemoApplication.NewAddress + "/index/index/page_show?type=user_agreement");
                context. startActivity(intent);
            }

            @Override
            public void updateDrawState(@NonNull TextPaint ds) {
                super.updateDrawState(ds);
                ds.setColor(color);
                ds.setUnderlineText(false);
            }
        }, start, end, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE);
        start = tips.indexOf(clickText2);
        end = start + clickText2.length();
        spannableString.setSpan(new ClickableSpan() {
            @Override
            public void onClick(@NonNull View widget) {
                Intent intent = new Intent(context, X5WebViewActivity.class);
                intent.putExtra("title", "隐私协议");
                intent.putExtra("url", DemoApplication.NewAddress + "/index/index/page_show?type=privacy_policy");
                context. startActivity(intent);
            }

            @Override
            public void updateDrawState(@NonNull TextPaint ds) {
                super.updateDrawState(ds);
                ds.setColor(color);
                ds.setUnderlineText(false);
            }
        }, start, end, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE);

        tvTips.setText(spannableString);
        tvTips.setMovementMethod(LinkMovementMethod.getInstance());
        leftText = findViewById(R.id.leftText);
        rightText = findViewById(R.id.rightText);
        leftText.setOnClickListener(left);
        rightText.setOnClickListener(right); 


    }


    private void setWidows() {
        WindowManager.LayoutParams lp = getWindow().getAttributes();
        getWindow().setGravity(Gravity.BOTTOM);
        lp.width = (int) (context.getResources().getDisplayMetrics().widthPixels * 0.90f);
        lp.height = WindowManager.LayoutParams.WRAP_CONTENT;
        lp.y = (int) (24 * context.getResources().getDisplayMetrics().density);
        lp.alpha = 1.0f;
        getWindow().setAttributes(lp);
        this.setCancelable(false);
        this.setCanceledOnTouchOutside(false);
    }

}
