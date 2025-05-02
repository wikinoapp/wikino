# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/settings/deletion", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record, :small)

    post "/s/#{space_record.identifier}/settings/deletion"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースに参加していないとき、404を返すこと" do
    space_record = create(:space_record, :small)
    other_space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: other_space_record, user_record:)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/settings/deletion"

    expect(response.status).to eq(404)
  end

  it "スペース名が一致しないとき、削除確認画面が再表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small, name: "テストスペース")
    create(:space_member_record, :owner, space_record:, user_record:)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/settings/deletion", params: {
      space_destroy_confirmation_form: {
        space_name: "異なるスペース名"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("スペース名が間違っています")
  end

  it "ログインしている & スペースのオーナーのとき、スペースが削除されてトップページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small, name: "テストスペース")
    create(:space_member_record, :owner, space_record:, user_record:)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/settings/deletion", params: {
      space_destroy_confirmation_form: {
        space_name: "テストスペース"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")
    expect(flash[:notice]).to eq("スペースを削除しました")

    expect(space_record.reload).to be_discarded
  end
end
