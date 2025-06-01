# typed: false
# frozen_string_literal: true

RSpec.describe "POST /user_session/two_factor_auth", type: :request do
  it "pending_user_idがセッションにないとき、ログインページにリダイレクトすること" do
    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ユーザーの2FAが無効なとき、ログインページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    set_session(pending_user_id: user_record.id)

    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "正しいTOTPコードのとき、ログインできること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    # 正しいTOTPコードを生成
    totp = ROTP::TOTP.new(two_factor_auth_record.secret)
    correct_code = totp.now

    expect(UserSessionRecord.count).to eq(0)

    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: correct_code
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # セッションが作成されていることを確認
    expect(UserSessionRecord.count).to eq(1)
    
    # pending_user_idがクリアされていることを確認
    expect(session[:pending_user_id]).to be_nil
  end

  it "正しいリカバリーコードのとき、ログインできること" do
    user_record = create(:user_record, :with_password)
    recovery_codes = ["code1234", "code5678", "code9012"]
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:, recovery_codes:)
    set_session(pending_user_id: user_record.id)

    expect(UserSessionRecord.count).to eq(0)

    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: "code1234"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # セッションが作成されていることを確認
    expect(UserSessionRecord.count).to eq(1)
    
    # 使用されたリカバリーコードが削除されていることを確認
    two_factor_auth_record.reload
    expect(two_factor_auth_record.recovery_codes).not_to include("code1234")
    expect(two_factor_auth_record.recovery_codes).to include("code5678")
    expect(two_factor_auth_record.recovery_codes).to include("code9012")
  end

  it "間違ったコードのとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: "000000"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("認証コードが正しくありません")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "コードが空のとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: ""
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("入力してください")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "既にログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    post "/user_session/two_factor_auth", params: {
      user_session_form_two_factor_verification: {
        code: "123456"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end
end