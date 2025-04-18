# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/pages/:page_number", type: :request do
  it "ログインしていない & 公開トピックのページのとき、ページが表示されること" do
    space = create(:space_record, :small)
    public_topic = create(:topic_record, :public, space_record: space)
    page = create(:page_record, space_record: space, topic_record: public_topic, title: "公開されているページ")

    get "/s/#{space.identifier}/pages/#{page.number}"
    page = Capybara.string(response.body)

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(page).to have_no_content("ゴミ箱に入れる")
  end

  it "ログインしていない & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space_record, :small)
    private_topic = create(:topic_record, :private, space_record: space)
    page = create(:page_record, space_record: space, topic_record: private_topic)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "別のスペースに参加している & 公開トピックのページのとき、ページが表示されること" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)
    public_topic = create(:topic_record, :public, space_record: space)
    page = create(:page_record, space_record: space, topic_record: public_topic, title: "公開されているページ")

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
  end

  it "別のスペースに参加している & 非公開トピックのページのとき、404を返すこと" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)
    private_topic = create(:topic_record, :private, space_record: space)
    page = create(:page_record, space_record: space, topic_record: private_topic)

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & 参加している公開トピックのページのとき、ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, :public, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)
    page = create(:page_record, space_record: space, topic_record: topic, title: "公開されているページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
  end

  it "スペースに参加している & 参加している非公開トピックのページのとき、ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, :private, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)
    page = create(:page_record, space_record: space, topic_record: topic, title: "公開されていないページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないページ")
  end

  it "スペースに参加している & 参加していない公開トピックのページのとき、ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, :public, space_record: space)
    page = create(:page_record, space_record: space, topic_record: topic, title: "公開されているページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
  end

  it "スペースに参加している & 参加していない非公開トピックのページのとき、ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, :private, space_record: space)
    page = create(:page_record, space_record: space, topic_record: topic, title: "公開されていないページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないページ")
  end
end
