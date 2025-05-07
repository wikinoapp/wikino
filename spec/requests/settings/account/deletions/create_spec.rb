# typed: false
# frozen_string_literal: true

RSpec.describe "POST /settings/account/deletion", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    post "/settings/account/deletion"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "アットネームが一致しないとき、削除確認画面が再表示されること" do
    user_record = create(:user_record, :with_password, atname: "test_user")

    sign_in(user_record:)

    post "/settings/account/deletion", params: {
      account_form_destroy_confirmation: {
        user_atname: "wrong_atname"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("アットネームが間違っています")
  end

  it "アクティブなスペースがあるとき、削除確認画面が再表示されること" do
    user_record = create(:user_record, :with_password, atname: "test_user")
    space_record = create(:space_record)
    create(:space_member_record, space_record:, user_record:)

    sign_in(user_record:)

    post "/settings/account/deletion", params: {
      account_form_destroy_confirmation: {
        user_atname: "test_user"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("アカウントにスペースが紐付いているため削除できません")
  end

  it "ログインしている & アクティブなスペースがないとき、ユーザーアカウントが削除されてホームページにリダイレクトすること" do
    user_record = create(:user_record, :with_password, atname: "test_user")

    sign_in(user_record:)

    post "/settings/account/deletion", params: {
      account_form_destroy_confirmation: {
        user_atname: "test_user"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to(root_path)
    expect(flash[:notice]).to eq("アカウントを削除しました")

    # ユーザーがdiscardされていることを確認
    expect(user_record.reload).to be_discarded

    # DestroyAccountJobがキューに入っていることを確認
    expect(DestroyAccountJob).to have_been_enqueued.with(user_record_id: user_record.id)
  end
end
