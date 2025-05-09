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

  it "アクセスしたページが紐付くトピックが削除されているとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    space_member_record = create(:space_member_record, space_record:, user_record:)

    topic_record = create(:topic_record, :public, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, space_record:, topic_record:, title: "テストページ")

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストページ")

    TopicService::SoftDestroy.new.call(topic_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

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

  it "スペースに参加している & ページがゴミ箱にあるとき、ページが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)

    page_record = create(:page_record, :trashed, {
      space_record:,
      title: "ゴミ箱にあるページ"
    })

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("ゴミ箱にあるページ")
    expect(response.body).to include("このページはゴミ箱に入れられています")
  end

  it "スペースに参加している & ゴミ箱にあるページにリンクしているとき、そのリンクは表示されないこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)

    linked_page_record = create(:page_record, :trashed, {
      space_record:,
      title: "リンクされているページ"
    })
    page_record = create(:page_record, {
      space_record:,
      title: "リンクしているページ",
      linked_page_ids: [linked_page_record.id]
    })

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("リンクされているページ")
  end

  it "スペースに参加している & 他のページをリンクしているページのとき、そのリンクが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)

    linked_page_record = create(:page_record, {
      space_record:,
      title: "リンクされているページ"
    })
    page_record = create(:page_record, {
      space_record:,
      title: "リンクしているページ",
      linked_page_ids: [linked_page_record.id]
    })

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("リンクされているページ")
  end

  it "スペースに参加している & ゴミ箱にあるページにリンクされているとき、そのバックリンクは表示されないこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)

    page_record = create(:page_record, space_record:, title: "リンクされているページ")
    create(:page_record, :trashed, {
      space_record:,
      title: "リンクしているページ",
      linked_page_ids: [page_record.id]
    })

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("リンクしているページ")
  end

  it "スペースに参加している & 他のページにリンクされているページのとき、バックリンクが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    create(:space_member_record, space_record:, user_record:)

    page_record = create(:page_record, space_record:, title: "リンクされているページ")
    create(:page_record, {
      space_record:,
      title: "リンクしているページ",
      linked_page_ids: [page_record.id]
    })

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/pages/#{page_record.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("リンクしているページ")
  end
end
