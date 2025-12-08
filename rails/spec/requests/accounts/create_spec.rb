# typed: false
# frozen_string_literal: true

RSpec.describe "POST /accounts", type: :request do
  def setup_email_confirmation
    expect(EmailConfirmationRecord.count).to eq(0)
    # 確認用コードを生成する
    post("/email_confirmation", params: {
      email_confirmations_creation_form: {
        email: "test@example.com"
      }
    })
    expect(EmailConfirmationRecord.count).to eq(1)

    EmailConfirmationRecord.first
  end

  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user_record, :with_password)
    sign_in(user_record: user)

    expect(UserRecord.count).to eq(1)

    post("/accounts", params: {
      accounts_creation_form: {
        atname: "test",
        password: "passw0rd"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # 新しいアカウントは作成されていないのでユーザーは1件のまま
    expect(UserRecord.count).to eq(1)
  end

  it "メールアドレスの確認に成功していないとき、トップページにリダイレクトすること" do
    email_confirmation = setup_email_confirmation
    # メールアドレスの確認に成功していない状態
    expect(email_confirmation.succeeded?).to be(false)

    expect(SpaceRecord.count).to eq(0)

    post("/accounts", params: {
      accounts_creation_form: {
        atname: "test",
        password: "passw0rd"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # 新しいアカウントは作成されていないのでスペースは0件のまま
    expect(SpaceRecord.count).to eq(0)
  end

  it "フォームの入力値に誤りがあるとき、エラーメッセージを表示すること" do
    email_confirmation = setup_email_confirmation
    # メールアドレスの確認が成功したことにする
    email_confirmation.success!

    expect(SpaceRecord.count).to eq(0)

    post("/accounts", params: {
      accounts_creation_form: {
        atname: "test",
        password: "1234" # 短すぎるパスワード
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("パスワードは8文字以上で入力してください")

    # エラーにより新しいアカウントは作成されていないのでスペースは0件のまま
    expect(SpaceRecord.count).to eq(0)
  end

  it "入力値が正しいとき、アカウントを作成してホーム画面にリダイレクトすること" do
    email_confirmation = setup_email_confirmation
    # メールアドレスの確認が成功したことにする
    email_confirmation.success!

    expect(UserRecord.count).to eq(0)

    post("/accounts", params: {
      accounts_creation_form: {
        atname: "test",
        password: "passw0rd"
      }
    })

    expect(response.status).to eq(302)

    # アカウントの作成に成功したのでユーザーが1件になる
    expect(UserRecord.count).to eq(1)

    expect(response).to redirect_to("/home")
  end
end
