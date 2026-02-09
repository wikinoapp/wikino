# typed: false
# frozen_string_literal: true

require "rotp"

RSpec.describe TwoFactorAuths::CreationForm, type: :form do
  it "パスワードが空文字列のとき、エラーになること" do
    user_record = create(:user_record)
    user_password_record = create(:user_password_record, user_id: user_record.id, password: "password123")
    user_two_factor_auth_record = create(:user_two_factor_auth_record,
      user_id: user_record.id,
      secret: "JBSWY3DPEHPK3PXP",
      enabled: false)
    user_record.update!(user_password_record:)
    user_record.update!(user_two_factor_auth_record:)

    form = TwoFactorAuths::CreationForm.new(password: "", totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("パスワードを入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: nil, totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("パスワードを入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: "a" * 7, totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("パスワードは8文字以上で入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: "wrongpassword", totp_code: "123456")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("パスワードが間違っています")
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

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("認証コードを入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: nil)
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("認証コードを入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "12345")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("認証コードは6文字で入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "1234567")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("認証コードは6文字で入力してください")
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

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "12345a")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("認証コードが間違っています")
  end

  it "user_recordがnilのとき、エラーにならないこと（他のバリデーションは実行される）" do
    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "123456")
    form.user_record = nil

    expect(form).to be_valid
  end

  it "2FAレコードが存在しないとき、エラーになること" do
    user_record_without_2fa = create(:user_record)
    create(:user_password_record, user_id: user_record_without_2fa.id, password: "password123")
    user_record_without_2fa.reload

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "123456")
    form.user_record = user_record_without_2fa

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("二要素認証が設定されていません")
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
    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: "000000")
    form.user_record = user_record

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("認証コードが間違っています")
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

    form = TwoFactorAuths::CreationForm.new(password: "password123", totp_code: valid_code)
    form.user_record = user_record

    expect(form).to be_valid
  end
end
