# typed: false
# frozen_string_literal: true

RSpec.describe "POST /email_confirmation", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    # 何もしていないのでEmailConfirmationは0件のはず
    expect(EmailConfirmation.count).to eq(0)

    user = create(:user, :with_password)
    sign_in(user:)

    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "hello@example.com"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # EmailConfirmationのレコードを作る前にリダイレクトされるので0件のままのはず
    expect(EmailConfirmation.count).to eq(0)
  end

  it "入力値が間違っているとき、エラーメッセージを表示すること" do
    # 何もしていないのでEmailConfirmationは0件のはず
    expect(EmailConfirmation.count).to eq(0)

    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "helloexample.com" # メールアドレスが間違っている (@がない)
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("メールアドレスは不正な値です")

    # エラーになったので0件のままのはず
    expect(EmailConfirmation.count).to eq(0)
  end

  it "入力値が正しいとき、確認用コードの入力ページに遷移すること" do
    # 何もしていないのでEmailConfirmationは0件のはず
    expect(EmailConfirmation.count).to eq(0)

    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "hello@example.com"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/email_confirmation/edit")

    # 処理に成功したのでEmailConfirmationが1件作られているはず
    expect(EmailConfirmation.count).to eq(1)

    # 作られたEmailConfirmationのIDがセッションに保存されているはず
    email_confirmation = EmailConfirmation.first
    expect(session[:email_confirmation_id]).to eq(email_confirmation.id)
  end
end
