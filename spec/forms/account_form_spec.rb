# typed: false
# frozen_string_literal: true

RSpec.describe AccountForm, type: :form do
  it "アットネームが空文字列のとき、エラーになること" do
    form = AccountForm.new(atname: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname can't be blank")
  end

  it "アットネームが `nil` のとき、エラーになること" do
    form = AccountForm.new(atname: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname can't be blank")
  end

  it "アットネームが21文字のとき、エラーになること" do
    form = AccountForm.new(atname: "a" * 21)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname is too long (maximum is 20 characters)")
  end

  it "アットネームが予約語のとき、エラーになること" do
    form = AccountForm.new(atname: "admin")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname cannot be used")
  end

  it "アットネームの形式が不正なとき、エラーになること" do
    form = AccountForm.new(atname: "a@b")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname is invalid")
  end

  it "アットネームがすでに使われているとき、エラーになること" do
    create(:user, atname: "a")
    form = AccountForm.new(atname: "a")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname has already been taken")
  end

  it "アットネームが1文字のとき、エラーにならないこと" do
    form = AccountForm.new(
      atname: "a",
      email: "test@example.com",
      locale: "ja",
      time_zone: "Asia/Tokyo",
      password: "password"
    )

    expect(form).to be_valid
  end

  it "アットネームが20文字のとき、エラーにならないこと" do
    form = AccountForm.new(
      atname: "a" * 20,
      email: "test@example.com",
      locale: "ja",
      time_zone: "Asia/Tokyo",
      password: "password"
    )

    expect(form).to be_valid
  end

  it "メールアドレスが空文字列のとき、エラーになること" do
    form = AccountForm.new(email: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Email can't be blank")
  end

  it "メールアドレスが `nil` のとき、エラーになること" do
    form = AccountForm.new(email: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Email can't be blank")
  end

  it "メールアドレスの形式が不正なとき、エラーになること" do
    form = AccountForm.new(email: "test")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Email is invalid")
  end

  it "メールアドレスがすでに使われているとき、エラーになること" do
    create(:user, email: "test@example.com")
    form = AccountForm.new(email: "test@example.com")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Email has already been taken")
  end

  it "ロケールが空文字列のとき、エラーになること" do
    form = AccountForm.new(locale: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Locale can't be blank")
  end

  it "ロケールが `nil` のとき、エラーになること" do
    form = AccountForm.new(locale: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Locale can't be blank")
  end

  it "ロケールの値が不正なとき、エラーになること" do
    form = AccountForm.new(locale: "invalid")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Locale is not included in the list")
  end

  it "パスワードが空文字列のとき、エラーになること" do
    form = AccountForm.new(password: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password can't be blank")
  end

  it "パスワードが `nil` のとき、エラーになること" do
    form = AccountForm.new(password: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password can't be blank")
  end

  it "パスワードが7文字のとき、エラーになること" do
    form = AccountForm.new(password: "a" * 7)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Password is too short (minimum is 8 characters)")
  end

  it "パスワードが8文字のとき、エラーにならないこと" do
    form = AccountForm.new(
      password: "a" * 8,
      atname: "a",
      email: "test@example.com",
      locale: "ja",
      time_zone: "Asia/Tokyo"
    )

    expect(form).to be_valid
  end

  it "タイムゾーンが空文字列のとき、エラーになること" do
    form = AccountForm.new(time_zone: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Time zone can't be blank")
  end

  it "タイムゾーンが `nil` のとき、エラーになること" do
    form = AccountForm.new(time_zone: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Time zone can't be blank")
  end

  it "タイムゾーンの形式が不正なとき、エラーになること" do
    form = AccountForm.new(time_zone: "invalid")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Time zone is invalid")
  end
end
