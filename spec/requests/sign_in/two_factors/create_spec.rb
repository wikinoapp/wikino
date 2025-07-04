# typed: false
# frozen_string_literal: true

RSpec.describe "POST /sign_in/two_factor", type: :request do
  it "既にログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` がセッションにないとき、ログインページにリダイレクトすること" do
    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` が無効なとき、ログインページにリダイレクトすること" do
    set_session(pending_user_id: "invalid_id")

    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` が有効なとき、ログインページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    set_session(pending_user_id: user_record.id)

    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & 間違った認証コードのとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: "000000"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("認証コードが間違っています")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & 認証コードが空のとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: ""
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("入力してください")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & 正しい認証コードのとき、ログインできること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    # 正しいTOTPコードを生成
    totp = ROTP::TOTP.new(two_factor_auth_record.secret)
    correct_code = totp.now

    expect(UserSessionRecord.count).to eq(0)

    post "/sign_in/two_factor", params: {
      user_session_form_two_factor_verification: {
        totp_code: correct_code
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # セッションが作成されていることを確認
    expect(UserSessionRecord.count).to eq(1)

    # pending_user_idがクリアされていることを確認
    expect(session[:pending_user_id]).to be_nil
  end
end
