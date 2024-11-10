# typed: false
# frozen_string_literal: true

RSpec.describe "POST /accounts", type: :request do
  def setup_email_confirmation
    expect(EmailConfirmation.count).to eq(0)
    # 確認用コードを生成する
    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "test@example.com"
      }
    })
    expect(EmailConfirmation.count).to eq(1)

    EmailConfirmation.first
  end

  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)
    sign_in(user:)

    expect(Space.count).to eq(1)

    post("/accounts", params: {
      account_form: {
        password: "passw0rd"
      }
    })

    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to("/s/#{space.identifier}")

    # 新しいアカウントは作成されていないのでスペースは1件のまま
    expect(Space.count).to eq(1)
  end

  it "メールアドレスの確認に成功していないとき、トップページにリダイレクトすること" do
    email_confirmation = setup_email_confirmation
    # メールアドレスの確認に成功していない状態
    expect(email_confirmation.succeeded?).to be(false)

    expect(Space.count).to eq(0)

    post("/accounts", params: {
      account_form: {
        password: "passw0rd"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # 新しいアカウントは作成されていないのでスペースは0件のまま
    expect(Space.count).to eq(0)
  end

  it "フォームの入力値に誤りがあるとき、エラーメッセージを表示すること" do
    email_confirmation = setup_email_confirmation
    # メールアドレスの確認が成功したことにする
    email_confirmation.success!

    expect(Space.count).to eq(0)

    post("/accounts", params: {
      account_form: {
        password: "1234" # 短すぎるパスワード
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("パスワードは8文字以上で入力してください")

    # エラーにより新しいアカウントは作成されていないのでスペースは0件のまま
    expect(Space.count).to eq(0)
  end

  it "入力値が正しいとき、アカウントを作成してホーム画面にリダイレクトすること" do
    email_confirmation = setup_email_confirmation
    # メールアドレスの確認が成功したことにする
    email_confirmation.success!

    expect(Space.count).to eq(0)

    post("/accounts", params: {
      account_form: {
        password: "passw0rd"
      }
    })

    expect(response.status).to eq(302)

    # アカウントの作成に成功したのでスペースが1件になる
    expect(Space.count).to eq(1)
    space = Space.first

    expect(response).to redirect_to("/s/#{space.identifier}")
  end
end
