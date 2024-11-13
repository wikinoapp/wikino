# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/pages/:page_number/links", type: :request do
  it "ログインしていない & 公開トピックのページのとき、ページのリンクが表示されること" do
    space = create(:space, :small)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)

    page_1 = create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    page_2 = create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")
    page = create(:page, space:, topic: public_topic, linked_page_ids: [page_1.id, page_2.id])

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    # 非公開トピックのページは表示されないはず
    expect(response.body).not_to include("公開されていないページ")
  end

  it "ログインしていない & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space, :small)
    private_topic = create(:topic, :private, space:)
    page = create(:page, space:, topic: private_topic)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(404)
  end

  it "別のスペースにログインしている & 公開トピックのページのとき、ページのリンクが表示されること" do
    space = create(:space, :small)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)

    page_1 = create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    page_2 = create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")
    page = create(:page, space:, topic: public_topic, linked_page_ids: [page_1.id, page_2.id])

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

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

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(404)
  end

  it "ログインしているとき、ページのリンクが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    not_joined_topic = create(:topic, space:)

    create(:topic_membership, space:, topic: public_topic, member: user)
    create(:topic_membership, space:, topic: private_topic, member: user)

    page_1 = create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    page_2 = create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")
    page_3 = create(:page, :published, space:, topic: not_joined_topic, title: "参加していないトピックのページ")
    page = create(:page, space:, linked_page_ids: [page_1.id, page_2.id, page_3.id])

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/links"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    # トピックに参加していなくてもページを見ることはできるはず
    expect(response.body).to include("参加していないトピックのページ")
  end
end
