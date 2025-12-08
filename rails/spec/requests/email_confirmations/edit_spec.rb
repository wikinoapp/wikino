# typed: false
# frozen_string_literal: true

RSpec.describe "GET /email_confirmation/edit", type: :request do
  it "EmailConfirmationのIDがセッションに格納されていないとき、トップページにリダイレクトすること" do
    get("/email_confirmation/edit")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")
  end

  it "EmailConfirmationのIDがセッションに格納されているとき、確認用コードの入力ページが表示されること" do
    email_confirmation = create(:email_confirmation_record)

    # EmailConfirmationのIDをセッションに格納する
    post("/email_confirmation", params: {
      email_confirmations_creation_form: {
        email: email_confirmation.email
      }
    })

    get("/email_confirmation/edit")

    expect(response.status).to eq(200)
    expect(response.body).to include("確認用コード入力")
  end
end
