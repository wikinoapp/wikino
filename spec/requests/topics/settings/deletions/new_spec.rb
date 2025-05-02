# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/settings/deletion/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record, :small)
    topic_record = create(:topic_record, space_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion/new"

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

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion/new"

    expect(response.status).to eq(404)
  end

  it "トピックに参加していないとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion/new"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックの管理者のとき、削除確認画面が表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:, name: "テストトピック")
    create(:topic_member_record, space_record:, topic_record:, space_member_record:, role: :admin)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/deletion/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストトピック")
    expect(response.body).to include("トピックを削除")
  end
end
