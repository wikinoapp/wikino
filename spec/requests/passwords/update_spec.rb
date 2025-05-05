# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /password", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    patch "/password"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "メールアドレスの確認が完了していないとき、トップページにリダイレクトすること" do
    get "/password/edit"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")
  end

  it "メールアドレスの確認が完了している & 入力値が不正なとき、エラーメッセージを表示すること" do
    email_confirmation = create(:email_confirmation_record, :succeeded, {
      email: "test@example.com",
      event: EmailConfirmationEvent::PasswordReset.serialize
    })
    set_session(email_confirmation_id: email_confirmation.id)

    patch("/password", params: {
      password_reset_form_creation: {
        password: "" # パスワードが空
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("パスワードを入力してください")
  end

  it "メールアドレスの確認が完了している & 入力値が正しいとき、パスワードがリセットされること" do
    user_record = create(:user_record, :with_password, email: "test@example.com")
    email_confirmation = create(:email_confirmation_record, :succeeded, {
      email: "test@example.com",
      event: EmailConfirmationEvent::PasswordReset.serialize
    })
    set_session(email_confirmation_id: email_confirmation.id)

    patch("/password", params: {
      password_reset_form_creation: {
        password: "new-password"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
    expect(flash[:notice]).to include("パスワードをリセットしました")

    # パスワードが更新されていることを確認
    user_record.reload
    expect(user_record.user_password_record.authenticate("new-password")).to be_truthy
  end
end
