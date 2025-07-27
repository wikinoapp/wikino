# typed: false
# frozen_string_literal: true

RSpec.describe "POST /password_reset", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    sign_in(user_record:)

    post "/password_reset", params: {
      email_confirmations_creation_form: {
        email: "test@example.com"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "無効なメールアドレスを入力した場合、エラーが表示されること" do
    expect(EmailConfirmationRecord.count).to eq(0)

    post "/password_reset", params: {
      email_confirmations_creation_form: {
        email: "invalid-email"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("メールアドレスを入力してください")
    expect(EmailConfirmationRecord.count).to eq(0)
  end

  it "有効なメールアドレスを入力した場合、メール確認画面にリダイレクトすること" do
    expect(EmailConfirmationRecord.count).to eq(0)

    valid_email = "test@example.com"

    post "/password_reset", params: {
      email_confirmations_creation_form: {
        email: valid_email
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to(edit_email_confirmation_path)
    expect(flash[:notice]).to eq("確認用のメールを送信しました")

    expect(EmailConfirmationRecord.count).to eq(1)
    email_confirmation = EmailConfirmationRecord.first
    expect(email_confirmation.email).to eq(valid_email)
    expect(email_confirmation.event_password_reset?).to be true
  end
end
