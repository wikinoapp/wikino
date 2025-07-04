# typed: false
# frozen_string_literal: true

require "rotp"

RSpec.describe TwoFactorAuthForm::Creation, type: :form do
  it "パスワードが空文字列のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "", totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password can't be blank")
  end

  it "パスワードが `nil` のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: nil, totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password can't be blank")
  end

  it "パスワードが7文字のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "a" * 7, totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password is too short (minimum is 8 characters)")
  end

  it "パスワードが間違っているとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "wrongpassword", totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password is incorrect")
  end

  it "TOTPコードが空文字列のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Authentication code can't be blank")
  end

  it "TOTPコードが `nil` のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: nil)
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Authentication code can't be blank")
  end

  it "TOTPコードが5文字のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "12345")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Authentication code is the wrong length (should be 6 characters)")
  end

  it "TOTPコードが7文字のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "1234567")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Authentication code is the wrong length (should be 6 characters)")
  end

  it "TOTPコードに数字以外が含まれているとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "12345a")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Authentication code is invalid")
  end

  it "user_recordがnilのとき、エラーにならないこと（他のバリデーションは実行される）" do
    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "123456")
    form.user_record = nil

    expect(form).to be_valid
  end

  it "2FAレコードが存在しないとき、エラーになること" do
    user_record_without_2fa = create(:user_record)
    create(:user_password_record, user_id: user_record_without_2fa.id, password: "password123")
    user_record_without_2fa.reload

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "123456")
    form.user_record = user_record_without_2fa

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Two-factor authentication is not set up")
  end

  it "TOTPコードが間違っているとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    # 固定のsecretを使用してテストする
    secret = "JBSWY3DPEHPK3PXP"
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: secret,
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    # 間違ったTOTPコードを使用
    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: "000000")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Authentication code is incorrect")
  end

  it "正しいパスワードとTOTPコードのとき、エラーにならないこと" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    # 固定のsecretを使用してテストする
    secret = "JBSWY3DPEHPK3PXP"
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: secret,
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    # 正しいTOTPコードを生成
    totp = ROTP::TOTP.new(secret)
    valid_code = totp.now

    form = TwoFactorAuthForm::Creation.new(password: "password123", totp_code: valid_code)
    form.user_record = user_record

    expect(form).to be_valid
  end
end
