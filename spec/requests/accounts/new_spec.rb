# typed: false
# frozen_string_literal: true

RSpec.describe "GET /accounts/new", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)
    sign_in(user:)

    get "/accounts/new"

    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to("/s/#{space.identifier}")
  end

  it "ログインしている & `skip_no_authentication` が付与されている & メールアドレスの確認に成功しているとき、アカウント作成ページが表示されること" do
    user = create(:user, :with_password)
    sign_in(user:)

    expect(EmailConfirmation.count).to eq(0)

    # 確認用コードを生成する
    post("/email_confirmation?skip_no_authentication=true", params: {
      new_email_confirmation_form: {
        email: "test@example.com"
      }
    })

    expect(EmailConfirmation.count).to eq(1)
    email_confirmation = EmailConfirmation.first
    # メールアドレスの確認が成功したことにする
    email_confirmation.success!

    get "/accounts/new?skip_no_authentication=true"

    expect(response.status).to eq(200)
    expect(response.body).to include("アカウント情報の登録")
  end

  it "メールアドレスの確認に成功しているとき、アカウント作成ページが表示されること" do
    expect(EmailConfirmation.count).to eq(0)

    # 確認用コードを生成する
    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "test@example.com"
      }
    })

    expect(EmailConfirmation.count).to eq(1)
    email_confirmation = EmailConfirmation.first
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
