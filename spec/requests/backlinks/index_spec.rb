# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/pages/:page_number/backlinks", type: :request do
  it "ログインしていない & 公開トピックのページのとき、ページのバックリンクが表示されること" do
    space = create(:space, :small)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)

    page = create(:page, space:, topic: public_topic)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ", linked_page_ids: [page.id])
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ", linked_page_ids: [page.id])

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "ログインしていない & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space, :small)
    private_topic = create(:topic, :private, space:)
    page = create(:page, space:, topic: private_topic)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(404)
  end

  it "別のスペースにログインしている & 公開トピックのページのとき、ページのバックリンクが表示されること" do
    space = create(:space, :small)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)

    page = create(:page, space:, topic: public_topic)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ", linked_page_ids: [page.id])
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ", linked_page_ids: [page.id])

    user = create(:user, :with_password)
    other_space = create(:space)
    create(:space_member, space: other_space, user:)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースに参加している & 非公開トピックのページのとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    private_topic = create(:topic, :private, space:)
    page = create(:page, space:, topic: private_topic)

    other_space = create(:space)
    create(:space_member, user:, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、ページのバックリンクが表示されること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    space_member = create(:space_member, space:, user:)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    not_joined_topic = create(:topic, space:)

    create(:topic_member, space:, topic: public_topic, space_member:)
    create(:topic_member, space:, topic: private_topic, space_member:)

    page = create(:page, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ", linked_page_ids: [page.id])
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ", linked_page_ids: [page.id])
    create(:page, :published, space:, topic: not_joined_topic, title: "参加していないトピックのページ", linked_page_ids: [page.id])

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    # トピックに参加していなくてもページを見ることはできるはず
    expect(response.body).to include("参加していないトピックのページ")
  end
end
