# typed: false
# frozen_string_literal: true

RSpec.describe "GET /accounts/new", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user_record, :with_password)
    sign_in(user_record: user)

    get "/accounts/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "メールアドレスの確認に成功しているとき、アカウント作成ページが表示されること" do
    expect(EmailConfirmationRecord.count).to eq(0)

    # 確認用コードを生成する
    post("/email_confirmation", params: {
      email_confirmation_form_creation: {
        email: "test@example.com"
      }
    })

    expect(EmailConfirmationRecord.count).to eq(1)
    email_confirmation = EmailConfirmationRecord.first
    # メールアドレスの確認が成功したことにする
    email_confirmation.success!

    get "/accounts/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("アカウント情報の登録")
  end

  it "メールアドレスの確認に成功していないとき、トップページにリダイレクトされること" do
    get "/accounts/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")
  end
end
