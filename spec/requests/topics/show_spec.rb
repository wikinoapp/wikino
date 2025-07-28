# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number", type: :request do
  it "ログインしていない & 非公開トピックのとき、404を返すこと" do
    space = create(:space_record, :small)
    topic = create(:topic_record, :private, space_record: space)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(404)
  end

  it "別のスペースに参加している & 非公開トピックのとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    topic = create(:topic_record, :private, space_record: space)

    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(404)
  end

  it "トピックが削除されているとき、404を返すこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:, name: "テストトピック")

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストトピック")

    Topics::SoftDestroyService.new.call(topic_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}"

    expect(response.status).to eq(404)
  end

  it "ページがゴミ箱にあるとき、そのページは表示されないこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:)
    create(:page_record, :published, :trashed, space_record:, topic_record:, title: "テストページ")

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ログインしていない & 公開トピックのとき、ページが表示されること" do
    space = create(:space_record, :small)
    topic = create(:topic_record, :public, space_record: space, name: "公開されているトピック")

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "別のスペースに参加している & 公開トピックのとき、ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    topic = create(:topic_record, :public, space_record: space, name: "公開されているトピック")

    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "スペースに参加している & 参加している公開トピックのとき、ページが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    space_member = create(:space_member_record, :owner, space_record: space, user_record: user)
    topic = create(:topic_record, :public, space_record: space, name: "公開されているトピック")
    create(:topic_member_record, topic_record: topic, space_member_record: space_member)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "スペースに参加している & 参加している非公開トピックのとき、ページが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    space_member = create(:space_member_record, :owner, space_record: space, user_record: user)
    topic = create(:topic_record, :private, space_record: space, name: "公開されていないトピック")
    create(:topic_member_record, topic_record: topic, space_member_record: space_member)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないトピック")
  end

  it "スペースに参加している & 参加していない公開トピックのとき、ページが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: space, user_record: user)
    topic = create(:topic_record, :public, space_record: space, name: "公開されているトピック")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "スペースに参加している & 参加していない非公開トピックのとき、ページが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: space, user_record: user)
    topic = create(:topic_record, :private, space_record: space, name: "公開されていないトピック")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないトピック")
  end
end
