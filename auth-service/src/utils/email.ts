import { Resend } from 'resend';
import { debugLog } from './debug-logger';

// Email configuration
const resend = new Resend(process.env.RESEND_API_KEY);

// Security alert email content
const getSecurityAlertContent = (language: string = 'en') => {
  const translations = {
    en: {
      subject: '🚨 Security Alert: New Device Login Attempt',
      greeting: 'Hi',
      message: 'We detected a login attempt from a new device on your account.',
      deviceInfo: 'Device Information:',
      deviceName: 'Device',
      platform: 'Platform',
      ipAddress: 'IP Address',
      time: 'Time',
      warning: 'If this was you, you can verify the device using the OTP sent to your email.',
      notYou: 'If this wasn\'t you:',
      action1: 'Change your password immediately',
      action2: 'Review your account activity',
      action3: 'Enable two-factor authentication if not already enabled',
      footer: 'This is an automated security alert. Please do not reply to this email.'
    },
    ar: {
      subject: '🚨 تنبيه أمني: محاولة تسجيل دخول من جهاز جديد',
      greeting: 'مرحباً',
      message: 'اكتشفنا محاولة تسجيل دخول من جهاز جديد على حسابك.',
      deviceInfo: 'معلومات الجهاز:',
      deviceName: 'الجهاز',
      platform: 'المنصة',
      ipAddress: 'عنوان IP',
      time: 'الوقت',
      warning: 'إذا كان هذا أنت، يمكنك التحقق من الجهاز باستخدام رمز التحقق المرسل إلى بريدك الإلكتروني.',
      notYou: 'إذا لم يكن هذا أنت:',
      action1: 'قم بتغيير كلمة المرور فوراً',
      action2: 'راجع نشاط حسابك',
      action3: 'قم بتفعيل المصادقة الثنائية إذا لم تكن مفعلة',
      footer: 'هذا تنبيه أمني آلي. يرجى عدم الرد على هذا البريد الإلكتروني.'
    }
  };
  return translations[language as keyof typeof translations] || translations.en;
};

// Multilingual email content
const getEmailContent = (type: 'verification' | 'passwordReset' | 'deviceVerification', language: string = 'en') => {
  const translations = {
    en: {
      verification: {
        subject: 'Verify Your Email Address',
        greeting: 'Hi',
        message: 'Thank you for registering! Please use the OTP below to verify your email address:',
        expires: 'This OTP will expire in 15 minutes.',
        ignore: 'If you didn\'t create an account, please ignore this email.',
        security: '',
        footer: 'This is an automated message from your Auth Service. Please do not reply to this email.'
      },
      passwordReset: {
        subject: 'Reset Your Password',
        greeting: 'Hi',
        message: 'We received a request to reset your password. Use the OTP below to proceed:',
        expires: 'This OTP will expire in 15 minutes.',
        ignore: 'If you didn\'t request a password reset, please ignore this email and your password will remain unchanged.',
        security: '',
        footer: 'This is an automated message from your Auth Service. Please do not reply to this email.'
      },
      deviceVerification: {
        subject: 'New Device Verification',
        greeting: 'Hi',
        message: 'We detected a new device trying to access your account. For your security, please verify this device using the OTP below:',
        expires: 'This OTP will expire in 15 minutes.',
        ignore: 'If you don\'t recognize this device, please ignore this email and secure your account.',
        security: 'Security Notice: If you don\'t recognize this activity, please secure your account immediately.',
        footer: 'This is an automated message from your Auth Service. Please do not reply to this email.'
      }
    },
    ar: {
      verification: {
        subject: 'تحقق من بريدك الإلكتروني',
        greeting: 'مرحباً',
        message: 'شكراً لتسجيلك! يرجى استخدام رمز التحقق أدناه للتحقق من عنوان بريدك الإلكتروني:',
        expires: 'ينتهي صلاحية رمز التحقق هذا خلال 15 دقيقة.',
        ignore: 'إذا لم تقم بإنشاء حساب، يرجى تجاهل هذا البريد الإلكتروني.',
        security: '',
        footer: 'هذه رسالة آلية من خدمة المصادقة. يرجى عدم الرد على هذا البريد الإلكتروني.'
      },
      passwordReset: {
        subject: 'إعادة تعيين كلمة المرور',
        greeting: 'مرحباً',
        message: 'استلمنا طلباً لإعادة تعيين كلمة المرور. استخدم رمز التحقق أدناه للمتابعة:',
        expires: 'ينتهي صلاحية رمز التحقق هذا خلال 15 دقيقة.',
        ignore: 'إذا لم تطلب إعادة تعيين كلمة المرور، يرجى تجاهل هذا البريد الإلكتروني وستبقى كلمة المرور الخاصة بك دون تغيير.',
        security: '',
        footer: 'هذه رسالة آلية من خدمة المصادقة. يرجى عدم الرد على هذا البريد الإلكتروني.'
      },
      deviceVerification: {
        subject: 'التحقق من جهاز جديد',
        greeting: 'مرحباً',
        message: 'اكتشفنا جهازاً جديداً يحاول الوصول إلى حسابك. لأمانك، يرجى التحقق من هذا الجهاز باستخدام رمز التحقق أدناه:',
        expires: 'ينتهي صلاحية رمز التحقق هذا خلال 15 دقيقة.',
        ignore: 'إذا لم تتعرف على هذا الجهاز، يرجى تجاهل هذا البريد الإلكتروني وتأمين حسابك.',
        security: 'ملاحظة أمنية: إذا لم تتعرف على هذا النشاط، يرجى تأمين حسابك فوراً.',
        footer: 'هذه رسالة آلية من خدمة المصادقة. يرجى عدم الرد على هذا البريد الإلكتروني.'
      }
    },
  };

  return translations[language as keyof typeof translations]?.[type] || translations.en[type];
};

// HTML escape helper to prevent injection
const escapeHtml = (value: string) =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

// Email templates
const getEmailTemplate = (type: 'verification' | 'passwordReset' | 'deviceVerification', otp: string, language: string = 'en', userName?: string) => {
  const content = getEmailContent(type, language);
  const isRTL = language === 'ar';

  const otpColor = type === 'passwordReset' ? '#dc3545' : type === 'deviceVerification' ? '#856404' : '#007bff';
  const bgColor = type === 'deviceVerification' ? '#fff3cd' : '#f8f9fa';
  const borderStyle = type === 'deviceVerification' ? 'border: 1px solid #ffeaa7;' : '';

  return {
    subject: content.subject,
    html: `
      <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; direction: ${isRTL ? 'rtl' : 'ltr'};">
        <h2 style="color: #333; text-align: center;">${type === 'verification' ? (language === 'en' ? 'Email Verification' : language === 'ar' ? 'التحقق من البريد الإلكتروني' : 'Vérification e-mail') : type === 'passwordReset' ? (language === 'en' ? 'Password Reset' : language === 'ar' ? 'إعادة تعيين كلمة المرور' : 'Réinitialisation mot de passe') : (language === 'en' ? 'Device Verification' : language === 'ar' ? 'التحقق من الجهاز' : 'Vérification appareil')}</h2>
        <p>${content.greeting} ${userName ? escapeHtml(userName) : (language === 'ar' ? '' : 'there')
      },</p>
        <p>${content.message}</p>
        <div style="background: ${bgColor}; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0; ${borderStyle}">
          <span style="font-size: 32px; font-weight: bold; letter-spacing: 3px; color: ${otpColor};">${otp}</span>
        </div>
        <p>${content.expires}</p>
        ${type === 'deviceVerification' ? `<p><strong>${content.security}</strong></p>` : ''}
        <p>${content.ignore}</p>
        <hr style="border: 1px solid #eee; margin: 30px 0;">
        <p style="color: #666; font-size: 12px; text-align: center;">
          ${content.footer}
        </p>
      </div>
    `,
  };
};

// Send email function
const sendEmail = async (to: string, subject: string, html: string): Promise<boolean> => {
  try {
    if (!process.env.RESEND_API_KEY) {
      debugLog('[Email] Resend API key missing. Skipping email send.');
      return false;
    }

    debugLog(`[Email] Attempting to send email to ${to} with subject: ${subject}`);
    const { data, error } = await resend.emails.send({
      from: process.env.EMAIL_FROM || 'onboarding@resend.dev',
      to: [to],
      subject,
      html,
    });

    if (error) {
      debugLog('[Email] Resend API error', { error: JSON.stringify(error) });
      return false;
    }

    debugLog(`[Email] ✓ Sent successfully to ${to} (ID: ${data?.id})`);
    return true;
  } catch (error) {
    debugLog('[Email] ✗ Failed to send email via Resend', { error: String(error) });
    return false;
  }
};

// Specific email sending functions
export const sendVerificationOTP = async (email: string, otp: string, userName?: string, language?: string): Promise<boolean> => {
  const template = getEmailTemplate('verification', otp, language, userName);
  return await sendEmail(email, template.subject, template.html);
};

export const sendPasswordResetOTP = async (email: string, otp: string, userName?: string, language?: string): Promise<boolean> => {
  const template = getEmailTemplate('passwordReset', otp, language, userName);
  return await sendEmail(email, template.subject, template.html);
};

export const sendDeviceVerificationOTP = async (email: string, otp: string, userName?: string, language?: string): Promise<boolean> => {
  const template = getEmailTemplate('deviceVerification', otp, language, userName);
  return await sendEmail(email, template.subject, template.html);
};

// Security alert email for new device login attempt
export const sendNewDeviceSecurityAlert = async (
  email: string,
  userName: string | undefined,
  deviceInfo: {
    name: string;
    platform: string | null;
    ipAddress: string | null;
  },
  language: string = 'en',
  otp?: string
): Promise<boolean> => {
  const content = getSecurityAlertContent(language);
  const isRTL = language === 'ar';
  const timestamp = new Date().toLocaleString(language === 'ar' ? 'ar-EG' : 'en-US', {
    dateStyle: 'full',
    timeStyle: 'short'
  });

  const otpBlock = otp ? `
    <div style="background: #e7f3ff; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0; border: 1px solid #b8daff;">
      <p style="margin-top: 0; color: #004085; font-size: 16px;"><strong>${isRTL ? 'رمز التحقق الخاص بك:' : 'Your Verification Code:'}</strong></p>
      <span style="font-size: 36px; font-weight: bold; letter-spacing: 5px; color: #007bff; display: block; margin: 10px 0;">${otp}</span>
      <p style="margin-bottom: 0; font-size: 12px; color: #666;">${isRTL ? 'ينتهي صلاحية هذا الرمز خلال 15 دقيقة.' : 'This code will expire in 15 minutes.'}</p>
    </div>
  ` : `<p style="color: #28a745;"><strong>${content.warning}</strong></p>`;

  const html = `
    <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; direction: ${isRTL ? 'rtl' : 'ltr'};">
      <div style="background: #dc3545; color: white; padding: 15px; border-radius: 8px 8px 0 0; text-align: center;">
        <h2 style="margin: 0;">🚨 ${content.subject.replace('🚨 ', '')}</h2>
      </div>
      
      <div style="border: 1px solid #dc3545; border-top: none; padding: 20px; border-radius: 0 0 8px 8px;">
        <p>${content.greeting} ${userName ? escapeHtml(userName) : (language === 'ar' ? '' : 'there')},</p>
        <p>${content.message}</p>
        
        <div style="background: #f8f9fa; padding: 15px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #dc3545;">
          <h4 style="margin-top: 0; color: #333;">${content.deviceInfo}</h4>
          <table style="width: 100%; border-collapse: collapse;">
            <tr>
              <td style="padding: 8px 0; color: #666;"><strong>${content.deviceName}:</strong></td>
              <td style="padding: 8px 0;">${escapeHtml(deviceInfo.name || 'Unknown')}</td>
            </tr>
            <tr>
              <td style="padding: 8px 0; color: #666;"><strong>${content.platform}:</strong></td>
              <td style="padding: 8px 0;">${escapeHtml(deviceInfo.platform || 'Unknown')}</td>
            </tr>
            <tr>
              <td style="padding: 8px 0; color: #666;"><strong>${content.ipAddress}:</strong></td>
              <td style="padding: 8px 0;">${escapeHtml(deviceInfo.ipAddress || 'Unknown')}</td>
            </tr>
            <tr>
              <td style="padding: 8px 0; color: #666;"><strong>${content.time}:</strong></td>
              <td style="padding: 8px 0;">${timestamp}</td>
            </tr>
          </table>
        </div>
        
        ${otpBlock}
        
        <div style="background: #fff3cd; padding: 15px; border-radius: 8px; margin: 20px 0; border: 1px solid #ffc107;">
          <p style="margin-top: 0; color: #856404;"><strong>${content.notYou}</strong></p>
          <ul style="color: #856404; margin-bottom: 0;">
            <li>${content.action1}</li>
            <li>${content.action2}</li>
            <li>${content.action3}</li>
          </ul>
        </div>
      </div>
      
      <hr style="border: 1px solid #eee; margin: 30px 0;">
      <p style="color: #666; font-size: 12px; text-align: center;">
        ${content.footer}
      </p>
    </div>
  `;

  return await sendEmail(email, content.subject, html);
};

export default resend;
