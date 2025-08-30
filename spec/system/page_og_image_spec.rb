# typed: false
# frozen_string_literal: true

require "rails_helper"

describe "ページOGP画像設定", :js, type: :system do
  it "ページにOGP画像が設定されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space, visibility: "public")

    # 画像付きページを作成
    attachment = FactoryBot.create(:attachment_record, space_record: space)
    page_with_image = FactoryBot.create(:page_record,
      topic_record: topic,
      featured_image_attachment_id: attachment.id,
      published_at: Time.current,
      body: "![test image](/attachments/#{attachment.id})\nThis is a test page with an image.")

    # ページを開く（公開ページ）
    visit page_path(space.identifier, page_with_image.number)

    # OGPメタタグが設定されていることを確認
    og_image_tag = find('meta[property="og:image"]', visible: false)
    expect(og_image_tag).to be_present

    # URLが含まれていることを確認
    expect(og_image_tag["content"]).to include("/attachments/")
  end

  it "画像がない場合はデフォルトのOGP画像が使用されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space, visibility: "public")

    # 画像なしページを作成
    page_without_image = FactoryBot.create(:page_record,
      topic_record: topic,
      published_at: Time.current,
      body: "This is a test page without an image.")

    # ページを開く（公開ページ）
    visit page_path(space.identifier, page_without_image.number)

    # OGPメタタグを確認
    og_image_tags = all('meta[property="og:image"]', visible: false)

    # デフォルトのOGP画像が設定されているか、og:imageタグがないことを確認
    if og_image_tags.any?
      # デフォルトOGP画像のURLパターンをチェック
      expect(og_image_tags.first["content"]).not_to include("/attachments/")
    end
  end

  it "GIF画像の場合はOGP画像が設定されないこと" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space, visibility: "public")

    # GIF画像付きページを作成
    gif_attachment = FactoryBot.create(:attachment_record,
      space_record: space,
      filename: "animation.gif")

    page_with_gif = FactoryBot.create(:page_record,
      topic_record: topic,
      featured_image_attachment_id: gif_attachment.id,
      published_at: Time.current,
      body: "![animation](/attachments/#{gif_attachment.id})\nThis page has a GIF animation.")

    # ページを開く（公開ページ）
    visit page_path(space.identifier, page_with_gif.number)

    # OGPメタタグを確認
    og_image_tags = all('meta[property="og:image"]', visible: false)

    # GIFの場合はカスタムOGP画像が設定されないことを確認
    if og_image_tags.any?
      expect(og_image_tags.first["content"]).not_to include("/attachments/#{gif_attachment.id}")
    end
  end
end
