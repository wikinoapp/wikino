# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/topics/:topic_number/settings/deletion", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record, :small)
    topic_record = create(:topic_record, space_record:)

    post "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースに参加していないとき、404を返すこと" do
    space_record = create(:space_record, :small)
    other_space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: other_space_record, user_record:)
    topic_record = create(:topic_record, space_record:)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion"

    expect(response.status).to eq(404)
  end

  it "トピックに参加していないとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion"

    expect(response.status).to eq(404)
  end

  it "トピック名が一致しないとき、削除確認画面が再表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:, name: "テストトピック")
    create(:topic_member_record, space_record:, topic_record:, space_member_record:, role: :admin)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion", params: {
      topic_destroy_confirmation_form: {
        topic_name: "異なるトピック名"
      }
    }

    expect(response.status).to eq(422)
    expect(response.body).to include("テストトピック")
  end

  it "ログインしている & スペースに参加している & トピックの管理者のとき、トピックが削除されてスペースのページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:, name: "テストトピック")
    create(:topic_member_record, space_record:, topic_record:, space_member_record:, role: :admin)

    sign_in(user_record:)

    post "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion", params: {
      topic_destroy_confirmation_form: {
        topic_name: "テストトピック"
      }
    }

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space_record.identifier}")
    expect(flash[:notice]).to eq("トピックを削除しました")

    expect(topic_record.reload).to be_discarded
  end
end
