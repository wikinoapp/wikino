# typed: false
# frozen_string_literal: true

RSpec.describe "Markup::AttachmentFilter", type: :model do
  before do
    # ActiveStorageのURL生成にホスト情報が必要
    Rails.application.routes.default_url_options[:host] = "test.host"
    # ActiveStorage::Current.url_optionsも設定
    ActiveStorage::Current.url_options = {host: "test.host"}
  end

  # ActiveStorage::Blobのurlメソッドをスタブ化するヘルパー
  def stub_blob_url(blob)
    # URLを直接モックする方法を変更
    signed_url = "http://test.host/rails/active_storage/blobs/redirect/#{blob.id}?expires_in=3600&signature=test_signature&space_member_id=test"
    allow(blob).to receive(:url).and_return(signed_url)
    # ActiveStorage::Blobクラス全体に対してもスタブを設定
    allow(ActiveStorage::Blob).to receive(:find).with(blob.id).and_return(blob)
  end

  def create_attachment(space:, filename:, content_type: "application/octet-stream")
    # ActiveStorageのBlobを作成
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("test content"),
      filename: filename,
      content_type: content_type
    )

    # Blobのurlメソッドをスタブ化
    stub_blob_url(blob)

    # ActiveStorageのAttachmentを作成
    user = FactoryBot.create(:user_record)
    space_member = FactoryBot.create(:space_member_record, space_record: space, user_record: user)

    # ActiveStorage::Attachmentを作成（実際のモデルに添付）
    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceMemberRecord",
      record_id: space_member.id,
      blob: blob
    )

    attachment = AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      space_record: space,
      active_storage_attachment_id: active_storage_attachment.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current
    )

    # AttachmentRecordのgenerate_signed_urlメソッドを直接スタブ化
    signed_url = "http://test.host/rails/active_storage/blobs/redirect/#{blob.id}?expires_in=3600&signature=test_signature&space_member_id=test"
    allow(attachment).to receive(:generate_signed_url).and_return(signed_url)

    # AttachmentRecord.find_byもスタブ化して、同じインスタンスを返すようにする
    allow(AttachmentRecord).to receive(:find_by).with(id: attachment.id, space_record: space).and_return(attachment)

    attachment
  end

  def render_markup(text:, current_topic:, current_space:, current_space_member: nil)
    markup = Markup.new(
      current_topic: current_topic,
      current_space: current_space,
      current_space_member: current_space_member
    )
    markup.render_html(text: text)
  end

  it "画像URLが署名付きURLに変換されること（メンバーの場合）" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    attachment = create_attachment(space: space_record, filename: "image.jpg")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    text = "![image](/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # 署名付きURLが生成されていることを確認
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")
    expect(output_html).to include("space_member_id")

    # 画像がa要素で囲まれていることを確認
    expect(output_html).to match(/<a[^>]*target="_blank"[^>]*>/)
    expect(output_html).to include("rel=\"noopener noreferrer\"")
    expect(output_html).to match(/<a[^>]*>.*<img[^>]*>.*<\/a>/m)
    expect(output_html).to include("max-w-full")
  end

  it "画像URLが署名付きURLに変換されること（非メンバーの場合でも）" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    attachment = create_attachment(space: space_record, filename: "image.jpg")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)

    text = "![image](/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: nil)

    # 署名付きURLが生成されていることを確認（非メンバーでも生成される）
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")

    # 画像がa要素で囲まれていることを確認
    expect(output_html).to match(/<a[^>]*target="_blank"[^>]*>/)
    expect(output_html).to include("rel=\"noopener noreferrer\"")
    expect(output_html).to match(/<a[^>]*>.*<img[^>]*>.*<\/a>/m)
    expect(output_html).to include("max-w-full")
  end

  it "インライン表示可能な画像形式の場合、img要素が維持されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    %w[jpg jpeg png gif svg webp].each do |ext|
      attachment = create_attachment(space: space_record, filename: "image.#{ext}")

      text = "![image](/attachments/#{attachment.id})"
      output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

      # img要素がa要素で囲まれていることを確認
      expect(output_html).to match(/<a[^>]*target="_blank"[^>]*>/)
      expect(output_html).to match(/<img[^>]*>/)
      expect(output_html).to include("src=")
      expect(output_html).to match(/<a[^>]*>.*<img[^>]*>.*<\/a>/m)
    end
  end

  it "インライン表示不可の形式の場合、ダウンロードリンクに変換されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    %w[pdf docx xlsx zip].each do |ext|
      attachment = create_attachment(space: space_record, filename: "document.#{ext}")

      text = "![document](/attachments/#{attachment.id})"
      output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

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
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    attachment = create_attachment(space: space_record, filename: "document.pdf")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    text = "[Download](/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # 署名付きURLが生成されていることを確認
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")

    # リンク属性が設定されていることを確認
    expect(output_html).to include("target=\"_blank\"")
    expect(output_html).to include("rel=\"noopener noreferrer\"")
  end

  it "異なるスペースの添付ファイルは変換されないこと" do
    space1_record = FactoryBot.create(:space_record)
    space2_record = FactoryBot.create(:space_record)
    topic1_record = FactoryBot.create(:topic_record, space_record: space1_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space1_record, user_record: user)
    attachment = create_attachment(space: space2_record, filename: "image.jpg")

    # space1でも添付ファイルを検索するので、そのスタブも追加
    allow(AttachmentRecord).to receive(:find_by).with(id: attachment.id, space_record: space1_record).and_return(nil)
    
    topic1 = TopicRepository.new.to_model(topic_record: topic1_record)
    space1 = SpaceRepository.new.to_model(space_record: space1_record)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    # space1のメンバーがspace2の添付ファイルにアクセスしようとする
    text = "![image](/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic1, current_space: space1, current_space_member: space_member)

    # URLが変換されないことを確認
    expect(output_html).not_to include("expires_in")
    expect(output_html).not_to include("signature")
  end

  it "存在しない添付ファイルIDの場合、変換されないこと" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    non_existent_id = SecureRandom.uuid
    text = "![image](/attachments/#{non_existent_id})"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # URLが変換されないことを確認
    expect(output_html).not_to include("expires_in")
    expect(output_html).not_to include("signature")
  end

  it "添付ファイルURLパターンにマッチしない場合、変換されないこと" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    # 通常の画像URL
    text = "![image](https://example.com/image.jpg)"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # URLが変更されていないことを確認
    expect(output_html).to include("https://example.com/image.jpg")
    expect(output_html).not_to include("expires_in")

    # 通常のリンク
    text = "[Link](https://example.com)"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    expect(output_html).to include("https://example.com")
    expect(output_html).not_to include("expires_in")
  end

  it "複数の添付ファイルが含まれる場合、すべて変換されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)

    attachment1 = create_attachment(space: space_record, filename: "image1.jpg")
    attachment2 = create_attachment(space: space_record, filename: "document.pdf")
    attachment3 = create_attachment(space: space_record, filename: "image2.png")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    text = <<~TEXT
      ![image1](/attachments/#{attachment1.id})

      Some text

      ![document](/attachments/#{attachment2.id})

      [Download](/attachments/#{attachment3.id})
    TEXT

    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # すべての添付ファイルURLが変換されていることを確認
    # attachment1は画像なのでa要素で囲まれたimg要素
    expect(output_html).to match(/<a[^>]*>.*<img[^>]*class="max-w-full".*<\/a>/m)

    # attachment2はPDFなのでリンクに変換
    expect(output_html).to include("document.pdf")
    expect(output_html).not_to match(/<img[^>]*src="[^"]*#{attachment2.id}[^"]*"/)

    # attachment3はリンクで署名付きURLに変換（署名付きURLが生成されている）
    expect(output_html).to include("Download")
    expect(output_html.scan(/<a[^>]*href="[^"]*expires_in[^"]*"/).size).to eq(3)
  end

  it "ファイル名に特殊文字が含まれる場合、適切にエスケープされること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)

    attachment = create_attachment(space: space_record, filename: "<script>alert('XSS')</script>.pdf")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    text = "![document](/attachments/#{attachment.id})"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # XSSが防がれていることを確認
    expect(output_html).not_to include("<script>alert('XSS')</script>")
    # CGI.escapeHTMLは < を - に変換しているようなので、ファイル名がエスケープされていることを確認
    expect(output_html).to include("-script-alert")
  end

  it "HTML形式のimg要素で挿入された画像も署名付きURLに変換されること（メンバーの場合）" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    attachment = create_attachment(space: space_record, filename: "image.jpg")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    # HTML形式のimg要素を含むテキスト
    text = "<img src=\"/attachments/#{attachment.id}\" alt=\"test image\">"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # 署名付きURLが生成されていることを確認
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")
    expect(output_html).to include("space_member_id")

    # 画像がa要素で囲まれていることを確認
    expect(output_html).to match(/<a[^>]*target="_blank"[^>]*>/)
    expect(output_html).to include("rel=\"noopener noreferrer\"")
    expect(output_html).to match(/<a[^>]*>.*<img[^>]*>.*<\/a>/m)
    expect(output_html).to include("max-w-full")
  end

  it "HTML形式のimg要素で挿入された画像も署名付きURLに変換されること（非メンバーの場合でも）" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    attachment = create_attachment(space: space_record, filename: "image.jpg")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)

    # HTML形式のimg要素を含むテキスト
    text = "<img src=\"/attachments/#{attachment.id}\" alt=\"test image\">"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: nil)

    # 署名付きURLが生成されていることを確認（非メンバーでも生成される）
    expect(output_html).to include("expires_in")
    expect(output_html).to include("signature")

    # 画像がa要素で囲まれていることを確認
    expect(output_html).to match(/<a[^>]*target="_blank"[^>]*>/)
    expect(output_html).to include("rel=\"noopener noreferrer\"")
    expect(output_html).to match(/<a[^>]*>.*<img[^>]*>.*<\/a>/m)
    expect(output_html).to include("max-w-full")
  end

  it "HTML形式のimg要素でインライン表示不可の形式の場合、ダウンロードリンクに変換されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)
    attachment = create_attachment(space: space_record, filename: "document.pdf")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    # HTML形式のimg要素を含むテキスト（PDFファイル）
    text = "<img src=\"/attachments/#{attachment.id}\" alt=\"pdf document\">"
    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # img要素ではなくリンク要素に変換されていることを確認
    expect(output_html).to match(/<a[^>]*>/)
    expect(output_html).to include("document.pdf")
    expect(output_html).to include("target=\"_blank\"")
    expect(output_html).to include("rel=\"noopener noreferrer\"")
    expect(output_html).to include("text-blue-600")

    # img要素が存在しないことを確認
    expect(output_html).not_to match(/<img[^>]*src="[^"]*#{attachment.id}[^"]*"/)

    # SVGアイコンが含まれていることを確認
    expect(output_html).to include("<svg")
  end

  it "複数のHTML形式img要素が含まれる場合、すべて変換されること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record: space_record)
    user = FactoryBot.create(:user_record)
    space_member_record = FactoryBot.create(:space_member_record, space_record: space_record, user_record: user)

    attachment1 = create_attachment(space: space_record, filename: "image1.jpg")
    attachment2 = create_attachment(space: space_record, filename: "image2.png")
    attachment3 = create_attachment(space: space_record, filename: "document.pdf")
    
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record:)
    space_member = SpaceMemberRepository.new.to_model(space_member_record:)

    text = <<~HTML
      <p>First image:</p>
      <img src="/attachments/#{attachment1.id}" alt="image1">
      <p>Second image:</p>
      <img src="/attachments/#{attachment2.id}" alt="image2">
      <p>PDF as image:</p>
      <img src="/attachments/#{attachment3.id}" alt="pdf">
    HTML

    output_html = render_markup(text: text, current_topic: topic, current_space: space, current_space_member: space_member)

    # 画像1と画像2は署名付きURLでa要素に囲まれていることを確認
    expect(output_html.scan(/<a[^>]*>.*?<img[^>]*class="max-w-full".*?>.*?<\/a>/m).size).to eq(2)

    # PDFはダウンロードリンクに変換されていることを確認
    expect(output_html).to include("document.pdf")
    expect(output_html).not_to match(/<img[^>]*src="[^"]*#{attachment3.id}[^"]*"/)

    # すべての添付ファイルが署名付きURLに変換されていることを確認
    expect(output_html.scan("expires_in").size).to be >= 3
  end
end
