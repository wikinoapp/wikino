# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe CardLinks::TopicWithActionComponent, type: :view do
  it "ページ作成権限がある場合、アクションボタンが表示されること" do
    # テストデータ作成
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: true
    )

    # コンポーネントをレンダリング
    render_inline(CardLinks::TopicWithActionComponent.new(topic:))

    # 検証：トピック名が表示される
    expect(page).to have_text(topic.name)

    # 検証：ページ作成リンクが表示される
    expect(page).to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/pages/new")

    # 検証：アイコンが表示される
    expect(page).to have_css("svg")
  end

  it "ページ作成権限がない場合、アクションボタンが非活性になること" do
    # テストデータ作成
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: false
    )

    # コンポーネントをレンダリング
    render_inline(CardLinks::TopicWithActionComponent.new(topic:))

    # 検証：トピック名が表示される
    expect(page).to have_text(topic.name)

    # 検証：ページ作成リンクがdisabled状態になる
    expect(page).to have_css("span.cursor-not-allowed.opacity-50")

    # 検証：リンクではなくspanタグになる
    expect(page).not_to have_link(href: "/s/#{space_record.identifier}/topics/#{topic_record.number}/pages/new")
  end

  it "トピックの説明が存在する場合に表示されること" do
    # テストデータ作成
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record,
      space_record:,
      description: "これはトピックの説明です")

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: true
    )

    # コンポーネントをレンダリング
    render_inline(CardLinks::TopicWithActionComponent.new(topic:))

    # 検証：説明文が表示される
    expect(page).to have_text("これはトピックの説明です")
  end

  it "カスタムクラスが適用されること" do
    # テストデータ作成
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    topic = TopicRepository.new.to_model(
      topic_record:,
      can_update: false,
      can_create_page: true
    )

    # コンポーネントをレンダリング
    render_inline(CardLinks::TopicWithActionComponent.new(topic:, card_class: "custom-class"))

    # 検証：カスタムクラスが適用される
    expect(page).to have_css(".custom-class")
  end
end
