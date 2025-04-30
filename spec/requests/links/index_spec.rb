# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/pages/:page_number/links", type: :request do
  it "トピックが削除されているとき、そのトピックに投稿されたページは表示されないこと" do
    space_record = create(:space_record)
    topic_record_1 = create(:topic_record, :public, space_record:)
    topic_record_2 = create(:topic_record, :public, space_record:)
    linked_page_record = create(:page_record, :published, space_record:, topic_record: topic_record_1, title: "テストページ")
    page_record = create(:page_record, space_record:, topic_record: topic_record_2, linked_page_ids: [linked_page_record.id])

    post "/s/#{space_record.identifier}/pages/#{page_record.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストページ")

    SoftDestroyTopicService.new.call(topic_record: topic_record_1)

    post "/s/#{space_record.identifier}/pages/#{page_record.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ログインしていない & 公開トピックのページのとき、ページのリンクが表示されること" do
    space = create(:space_record, :small)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)

    page_1 = create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    page_2 = create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")
    page = create(:page_record, space_record: space, topic_record: public_topic, linked_page_ids: [page_1.id, page_2.id])

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "ログインしていない & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space_record, :small)
    private_topic = create(:topic_record, :private, space_record: space)
    page = create(:page_record, space_record: space, topic_record: private_topic)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(404)
  end

  it "別のスペースに参加している & 公開トピックのページのとき、ページのリンクが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)

    page_1 = create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    page_2 = create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")
    page = create(:page_record, space_record: space, topic_record: public_topic, linked_page_ids: [page_1.id, page_2.id])

    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースに参加している & 非公開トピックのページのとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    private_topic = create(:topic_record, :private, space_record: space)
    page = create(:page_record, space_record: space, topic_record: private_topic)

    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、ページのリンクが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    not_joined_topic = create(:topic_record, space_record: space)

    create(:topic_member_record, space_record: space, topic_record: public_topic, space_member_record: space_member)
    create(:topic_member_record, space_record: space, topic_record: private_topic, space_member_record: space_member)

    page_1 = create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    page_2 = create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")
    page_3 = create(:page_record, :published, space_record: space, topic_record: not_joined_topic, title: "参加していないトピックのページ")
    page = create(:page_record, space_record: space, linked_page_ids: [page_1.id, page_2.id, page_3.id])

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    # トピックに参加していなくてもページを見ることはできるはず
    expect(response.body).to include("参加していないトピックのページ")
  end
end
