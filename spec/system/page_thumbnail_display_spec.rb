# typed: false
# frozen_string_literal: true

require "rails_helper"

describe "ページサムネイル表示", :js, type: :system do
  it "ページ一覧でサムネイル画像が表示されること" do
    # スペースとトピックを作成
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    # 画像付きページを作成
    FactoryBot.create(:page_record,
      topic_record: topic,
      body: "![test image](/attachments/test-attachment-id)\nThis is a test page with an image.")

    # 画像なしページを作成
    page_without_image = FactoryBot.create(:page_record,
      topic_record: topic,
      body: "This is a test page without an image.")

    # featured_image_attachment_idを持つページを作成
    attachment = FactoryBot.create(:attachment_record, space_record: space)
    page_with_featured = FactoryBot.create(:page_record,
      topic_record: topic,
      featured_image_attachment_id: attachment.id,
      body: "![test image](/attachments/#{attachment.id})\nThis page has a featured image.")

    # ログイン
    sign_in_as(user)

    # トピックページへ移動
    visit topic_path(space.identifier, topic.number)

    # 画像付きページのカードに画像が表示されることを確認
    within("[data-testid='page-card-#{page_with_featured.number}']") do
      expect(page).to have_css("img")
    end

    # 画像なしページのカードに画像が表示されないことを確認
    within("[data-testid='page-card-#{page_without_image.number}']") do
      expect(page).not_to have_css("img")
    end
  end

  it "GIF画像の場合はオリジナル画像が表示されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    # GIFアニメーション画像を持つページを作成
    gif_attachment = FactoryBot.create(:attachment_record,
      space_record: space,
      filename: "animation.gif")

    page_with_gif = FactoryBot.create(:page_record,
      topic_record: topic,
      featured_image_attachment_id: gif_attachment.id,
      body: "![animation](/attachments/#{gif_attachment.id})\nThis page has a GIF animation.")

    # ログイン
    sign_in_as(user)

    # トピックページへ移動
    visit topic_path(space.identifier, topic.number)

    # GIF画像が表示されることを確認
    within("[data-testid='page-card-#{page_with_gif.number}']") do
      img = find("img")
      # GIFの場合はオリジナル画像のURLが使用されることを確認
      expect(img["src"]).to include("/attachments/#{gif_attachment.id}")
    end
  end
end
