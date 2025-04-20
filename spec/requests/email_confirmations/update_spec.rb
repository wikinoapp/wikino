# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /email_confirmation", type: :request do
  it "EmailConfirmationのIDがセッションに格納されていないとき、トップページにリダイレクトすること" do
    patch("/email_confirmation", params: {
      email_confirmation_form: {
        confirmation_code: "123456"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")
  end

  it "確認用コードが間違っているとき、エラーメッセージを表示すること" do
    # 確認用コードを生成する
    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "test@example.com"
      }
    })

    patch("/email_confirmation", params: {
      email_confirmation_form: {
        confirmation_code: "wrong_code" # 間違った確認用コード
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("確認用コードが間違っているか古くなっています")
  end

  it "アカウント作成のメールアドレスの確認に成功したとき、アカウント作成ページにリダイレクトすること" do
    expect(EmailConfirmation.count).to eq(0)

    # 確認用コードを生成する
    post("/email_confirmation", params: {
      new_email_confirmation_form: {
        email: "test@example.com"
      }
    })

    expect(EmailConfirmation.count).to eq(1)
    email_confirmation = EmailConfirmation.first

    # 確認用コードの検証が済んでいないのでまだ成功していない
    expect(email_confirmation.succeeded?).to be(false)

    patch("/email_confirmation", params: {
      email_confirmation_form: {
        confirmation_code: email_confirmation.code
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/accounts/new")

    # 確認用コードの検証が済んだので成功している
    expect(email_confirmation.reload.succeeded?).to be(true)
  end
end
