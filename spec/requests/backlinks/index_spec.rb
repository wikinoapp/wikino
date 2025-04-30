# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/pages/:page_number/backlinks", type: :request do
  it "トピックが削除されているとき、そのトピックに投稿されたページは表示されないこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:)
    page_record = create(:page_record, space_record:, topic_record:)
    create(
      :page_record,
      :published,
      space_record:,
      topic_record:,
      title: "テストページ",
      linked_page_ids: [page_record.id]
    )

    post "/s/#{space_record.identifier}/pages/#{page_record.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストページ")

    SoftDestroyTopicService.new.call(topic_record:)

    post "/s/#{space_record.identifier}/pages/#{page_record.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ログインしていない & 公開トピックのページのとき、ページのバックリンクが表示されること" do
    space = create(:space_record, :small)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)

    page = create(:page_record, space_record: space, topic_record: public_topic)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ", linked_page_ids: [page.id])
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ", linked_page_ids: [page.id])

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "ログインしていない & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space_record, :small)
    private_topic = create(:topic_record, :private, space_record: space)
    page = create(:page_record, space_record: space, topic_record: private_topic)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(404)
  end

  it "別のスペースにログインしている & 公開トピックのページのとき、ページのバックリンクが表示されること" do
    space = create(:space_record, :small)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)

    page = create(:page_record, space_record: space, topic_record: public_topic)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ", linked_page_ids: [page.id])
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ", linked_page_ids: [page.id])

    user = create(:user_record, :with_password)
    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

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

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、ページのバックリンクが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    not_joined_topic = create(:topic_record, space_record: space)

    create(:topic_member_record, space_record: space, topic_record: public_topic, space_member_record: space_member)
    create(:topic_member_record, space_record: space, topic_record: private_topic, space_member_record: space_member)

    page = create(:page_record, space_record: space)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ", linked_page_ids: [page.id])
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ", linked_page_ids: [page.id])
    create(:page_record, :published, space_record: space, topic_record: not_joined_topic, title: "参加していないトピックのページ", linked_page_ids: [page.id])

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    # トピックに参加していなくてもページを見ることはできるはず
    expect(response.body).to include("参加していないトピックのページ")
  end
end
