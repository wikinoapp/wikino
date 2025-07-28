# typed: false
# frozen_string_literal: true

RSpec.describe "POST /sign_in/two_factor/recovery", type: :request do
  it "既にログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: "xxxxxxxx"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` がセッションにないとき、ログインページにリダイレクトすること" do
    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: "xxxxxxxx"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` が無効なとき、ログインページにリダイレクトすること" do
    set_session(pending_user_id: "invalid_id")

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: "xxxxxxxx"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` が有効なとき、ログインページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    set_session(pending_user_id: user_record.id)

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_verification_form: {
        recovery_code: "xxxxxxxx"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & 間違ったコードのとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: "xxxxxxxx"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("リカバリーコードが間違っています")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & コードが空のとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: ""
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("入力してください")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & 間違ったリカバリーコードのとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    recovery_codes = ["code1234", "code5678", "code9012"]
    create(:user_two_factor_auth_record, :enabled, user_record:, recovery_codes:)
    set_session(pending_user_id: user_record.id)

    expect(UserSessionRecord.count).to eq(0)

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: "wrongcode"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("リカバリーコードが間違っています")

    # セッションが作成されていないことを確認
    expect(UserSessionRecord.count).to eq(0)
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効 & 正しいリカバリーコードのとき、ログインできること" do
    user_record = create(:user_record, :with_password)
    recovery_codes = ["code1234", "code5678", "code9012"]
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:, recovery_codes:)
    set_session(pending_user_id: user_record.id)

    expect(UserSessionRecord.count).to eq(0)

    post "/sign_in/two_factor/recovery", params: {
      user_sessions_two_factor_recovery_form: {
        recovery_code: "code1234"
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
end
