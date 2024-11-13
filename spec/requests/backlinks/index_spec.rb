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

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースにログインしている & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space, :small)
    private_topic = create(:topic, :private, space:)
    page = create(:page, space:, topic: private_topic)

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/backlinks"

    expect(response.status).to eq(404)
  end

  it "ログインしているとき、ページのバックリンクが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    not_joined_topic = create(:topic, space:)

    create(:topic_membership, space:, topic: public_topic, member: user)
    create(:topic_membership, space:, topic: private_topic, member: user)

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
