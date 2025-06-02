# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /settings/two_factor_auth", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    delete "/settings/two_factor_auth"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & 2FAが無効なとき、二要素認証の設定ページにリダイレクトされること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    delete "/settings/two_factor_auth", params: {
      two_factor_auth_form_destruction: {
        password: "passw0rd"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth")
  end

  it "ログインしている & パスワードが間違っているとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    delete "/settings/two_factor_auth", params: {
      two_factor_auth_form_destruction: {
        password: "wrong_password"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("パスワードが間違っています")

    # 2FAが無効化されていないことを確認
    two_factor_auth_record.reload
    expect(two_factor_auth_record.enabled).to be true
  end

  it "ログインしている & 正しいパスワードのとき、2FAが無効化されること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    delete "/settings/two_factor_auth", params: {
      two_factor_auth_form_destruction: {
        password: "passw0rd"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth")

    # 2FAが無効化されていることを確認
    two_factor_auth_record.reload
    expect(two_factor_auth_record.enabled).to be false
    expect(two_factor_auth_record.enabled_at).to be_nil
  end
end
