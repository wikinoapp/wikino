# typed: false
# frozen_string_literal: true

RSpec.describe "Markup::AttachmentFilter", type: :model do
  def create_attachment(space:, filename:, content_type: "application/octet-stream")
    # ActiveStorageのBlobを作成
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("test content"),
      filename: filename,
      content_type: content_type
    )

    AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space: space,
      filename: filename,
      byte_size: blob.byte_size,
      content_type: content_type,
      blob_id: blob.id,
      status: "validated"
    )
  end

  def render_markup(text:, current_topic:, current_space_member: nil)
    markup = Markup.new(
      current_topic: current_topic,
      current_space_member: current_space_member
    )
    markup.render_html(text: text)
  end

  it "画像URLが署名付きURLに変換されること（メンバーの場合）" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)
    attachment = create_attachment(space: space, filename: "image.jpg")

    text = "![image](/s/#{space.identifier}/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    # 署名付きURLが生成されていることを確認
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")
    expect(output_html).to include("space_member_id")
    
    # 画像の属性が設定されていることを確認
    expect(output_html).to include("cursor-pointer")
    expect(output_html).to include("hover:opacity-90")
    expect(output_html).to include("data-controller=\"attachment-viewer\"")
    expect(output_html).to include("data-action=\"click->attachment-viewer#open\"")
  end

  it "画像URLが変換されないこと（非メンバーの場合）" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    attachment = create_attachment(space: space, filename: "image.jpg")

    text = "![image](/s/#{space.identifier}/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: nil)
    
    # 署名付きURLが生成されていないことを確認
    expect(output_html).not_to include("expires_in")
    expect(output_html).not_to include("signature")
    
    # インライン表示用の属性も設定されていないことを確認
    expect(output_html).not_to include("data-controller=\"attachment-viewer\"")
  end

  it "インライン表示可能な画像形式の場合、img要素が維持されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    %w[jpg jpeg png gif svg webp].each do |ext|
      attachment = create_attachment(space: space, filename: "image.#{ext}")

      text = "![image](/s/#{space.identifier}/attachments/#{attachment.id})"
      output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)
      
      # img要素が維持されていることを確認
      expect(output_html).to match(/<img[^>]*>/)
      expect(output_html).to include("src=")
    end
  end

  it "インライン表示不可の形式の場合、ダウンロードリンクに変換されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    %w[pdf docx xlsx zip].each do |ext|
      attachment = create_attachment(space: space, filename: "document.#{ext}")

      text = "![document](/s/#{space.identifier}/attachments/#{attachment.id})"
      output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)
      
      # リンク要素に変換されていることを確認
      expect(output_html).to match(/<a[^>]*>/)
      expect(output_html).to include("document.#{ext}")
      expect(output_html).to include("target=\"_blank\"")
      expect(output_html).to include("rel=\"noopener noreferrer\"")
      expect(output_html).to include("text-blue-600")
      
      # SVGアイコンが含まれていることを確認
      expect(output_html).to include("<svg")
    end
  end

  it "添付ファイルリンクが署名付きURLに変換されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)
    attachment = create_attachment(space: space, filename: "document.pdf")

    text = "[Download](/s/#{space.identifier}/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    # 署名付きURLが生成されていることを確認
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")
    
    # リンク属性が設定されていることを確認
    expect(output_html).to include("target=\"_blank\"")
    expect(output_html).to include("rel=\"noopener noreferrer\"")
  end

  it "異なるスペースの添付ファイルは変換されないこと" do
    space1 = FactoryBot.create(:space_record)
    space2 = FactoryBot.create(:space_record)
    topic1 = FactoryBot.create(:topic_record, space_record: space1)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space1, user_record: user)
    attachment = create_attachment(space: space2, filename: "image.jpg")

    # space1のメンバーがspace2の添付ファイルにアクセスしようとする
    text = "![image](/s/#{space2.identifier}/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic1, current_space_member: space_member)
    
    # URLが変換されないことを確認
    expect(output_html).not_to include("expires_in")
    expect(output_html).not_to include("signature")
  end

  it "存在しない添付ファイルIDの場合、変換されないこと" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    non_existent_id = SecureRandom.uuid
    text = "![image](/s/#{space.identifier}/attachments/#{non_existent_id})"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    # URLが変換されないことを確認
    expect(output_html).not_to include("expires_in")
    expect(output_html).not_to include("signature")
  end

  it "添付ファイルURLパターンにマッチしない場合、変換されないこと" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    # 通常の画像URL
    text = "![image](https://example.com/image.jpg)"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    # URLが変更されていないことを確認
    expect(output_html).to include("https://example.com/image.jpg")
    expect(output_html).not_to include("expires_in")

    # 通常のリンク
    text = "[Link](https://example.com)"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    expect(output_html).to include("https://example.com")
    expect(output_html).not_to include("expires_in")
  end

  it "複数の添付ファイルが含まれる場合、すべて変換されること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    attachment1 = create_attachment(space: space, filename: "image1.jpg")
    attachment2 = create_attachment(space: space, filename: "document.pdf")
    attachment3 = create_attachment(space: space, filename: "image2.png")

    text = <<~TEXT
      ![image1](/s/#{space.identifier}/attachments/#{attachment1.id})

      Some text

      ![document](/s/#{space.identifier}/attachments/#{attachment2.id})

      [Download](/s/#{space.identifier}/attachments/#{attachment3.id})
    TEXT

    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    # すべての添付ファイルURLが変換されていることを確認
    # attachment1は画像なのでimg要素のまま
    expect(output_html).to match(/<img[^>]*src="[^"]*#{attachment1.id}[^"]*"/)
    
    # attachment2はPDFなのでリンクに変換
    expect(output_html).to include("document.pdf")
    expect(output_html).not_to match(/<img[^>]*src="[^"]*#{attachment2.id}[^"]*"/)
    
    # attachment3はリンクで署名付きURLに変換
    expect(output_html).to match(/<a[^>]*href="[^"]*#{attachment3.id}[^"]*"/)
  end

  it "ファイル名に特殊文字が含まれる場合、適切にエスケープされること" do
    space = FactoryBot.create(:space_record)
    topic = FactoryBot.create(:topic_record, space_record: space)
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    attachment = create_attachment(space: space, filename: "<script>alert('XSS')</script>.pdf")

    text = "![document](/s/#{space.identifier}/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space_member: space_member)

    # XSSが防がれていることを確認
    expect(output_html).not_to include("<script>alert('XSS')</script>")
    expect(output_html).to include("&lt;script&gt;")
  end
end
