# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/trash", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースに参加していないとき、404を返すこと" do
    space = create(:space_record, :small)
    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(404)
  end

  it "トピックが削除されているとき、そのトピックに紐付くページは表示されないこと" do
    space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)
    create(:page_record, :trashed, space_record:, topic_record:, title: "削除されたページ")

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/trash"

    expect(response.status).to eq(200)
    expect(response.body).to include("削除されたページ")

    DestroyTopicService.new.call(topic_record:)

    get "/s/#{space_record.identifier}/trash"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("削除されたページ")
  end

  it "スペースに参加しているとき、ゴミ箱ページが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: space, user_record: user)
    topic = create(:topic_record, space_record: space)
    create(:page_record, :trashed, space_record: space, topic_record: topic, title: "削除されたページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(200)
    expect(response.body).to include("ゴミ箱に入れたページを表示しています。")
    expect(response.body).to include("削除されたページ")
  end
end
