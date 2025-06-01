# typed: false
# frozen_string_literal: true

RSpec.describe "POST /settings/two_factor_auth/recovery_codes", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    post "/settings/two_factor_auth/recovery_codes"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "2FAが無効なとき、404エラーになること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    post "/settings/two_factor_auth/recovery_codes", params: {
      two_factor_auth_form_recovery_code_regeneration: {
        password: "passw0rd"
      }
    }

    expect(response.status).to eq(404)
  end

  it "パスワードが間違っているとき、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)
    old_recovery_codes = two_factor_auth_record.recovery_codes.dup

    sign_in_with_2fa(user_record:)

    post "/settings/two_factor_auth/recovery_codes", params: {
      two_factor_auth_form_recovery_code_regeneration: {
        password: "wrong_password"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("パスワードが間違っています")

    # リカバリーコードが変更されていないことを確認
    two_factor_auth_record.reload
    expect(two_factor_auth_record.recovery_codes).to eq(old_recovery_codes)
  end

  it "正しいパスワードのとき、リカバリーコードが再生成されること" do
    user_record = create(:user_record, :with_password)
    two_factor_auth_record = create(:user_two_factor_auth_record, :enabled, user_record:)
    old_recovery_codes = two_factor_auth_record.recovery_codes.dup

    sign_in_with_2fa(user_record:)

    post "/settings/two_factor_auth/recovery_codes", params: {
      two_factor_auth_form_recovery_code_regeneration: {
        password: "passw0rd"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/two_factor_auth/recovery_codes")

    # リカバリーコードが変更されていることを確認
    two_factor_auth_record.reload
    expect(two_factor_auth_record.recovery_codes).not_to eq(old_recovery_codes)
    expect(two_factor_auth_record.recovery_codes.size).to eq(10)
    
    # 新しいリカバリーコードがセッションに設定されていることを確認はコントローラー側で実行されるためスキップ
  end
end