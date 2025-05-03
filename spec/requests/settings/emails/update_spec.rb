# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /settings/email", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトされること" do
    patch "/settings/email"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & 入力値が不正なとき、エラーメッセージを表示すること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    patch("/settings/email", params: {
      email_form_edit: {
        new_email: "" # メールアドレスが空
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("メールアドレスを入力してください")
  end

  it "ログインしている & 入力値が正しいとき、メールアドレス確認画面にリダイレクトされること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    patch("/settings/email", params: {
      email_form_edit: {
        new_email: "new@example.com"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/email_confirmation/edit")
  end
end
