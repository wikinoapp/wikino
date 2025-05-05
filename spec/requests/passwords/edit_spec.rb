# typed: false
# frozen_string_literal: true

RSpec.describe "GET /password/edit", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/password/edit"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "メールアドレスの確認が完了しているとき、パスワード設定画面が表示されること" do
    email_confirmation = create(:email_confirmation_record, :succeeded, {
      email: "test@example.com",
      event: EmailConfirmationEvent::PasswordReset.serialize
    })
    set_session(email_confirmation_id: email_confirmation.id)

    get "/password/edit"

    expect(response.status).to eq(200)
    expect(response.body).to include("新しいパスワードを入力してください")
  end
end
