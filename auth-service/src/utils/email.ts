import { Resend } from 'resend';
import dotenv from "dotenv";
dotenv.config();

// Email configuration
const resend = new Resend(process.env.RESEND_API_KEY);

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
        <p>${content.greeting} ${userName || (language === 'ar' ? '' : 'there')},</p>
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
      console.warn('Resend API key missing. Skipping email send.');
      return false;
    }

    const { data, error } = await resend.emails.send({
      from: process.env.EMAIL_FROM || 'onboarding@resend.dev',
      to: [to],
      subject,
      html,
    });
    if (error) {
      console.error('Resend API error:', error);
      return false;
    }

    console.log(`Email sent successfully to ${to} via Resend (ID: ${data?.id})`);
    return true;
  } catch (error) {
    console.error('Failed to send email via Resend:', error);
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

export default resend;
