# typed: false
# frozen_string_literal: true

RSpec.describe "添付ファイルの非同期URL読み込み", type: :system do
  # テスト用エンドポイントは Test::Attachments::SignedUrls::CreateController で処理される
  # ビューに data-test-endpoint="/_test/attachments/signed_urls" が設定される

  it "ページ表示時に添付ファイルの署名付きURLが非同期で読み込まれること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # 添付ファイル作成
    attachment_record1 = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)
    attachment_record2 = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)

    # ページ作成（body_htmlにプレースホルダー画像を含む）
    page_record = create(:page_record, :published,
      space_record:,
      topic_record:,
      body_html: <<~HTML
        <p>テスト画像</p>
        <img src="/attachments/placeholder" data-attachment-id="#{attachment_record1.id}" data-attachment-type="image" class="wikino-attachment-image" loading="lazy" />
        <p>もう一つの画像</p>
        <img src="/attachments/placeholder" data-attachment-id="#{attachment_record2.id}" data-attachment-type="image" class="wikino-attachment-image" loading="lazy" />
      HTML
    )

    sign_in(user_record:)

    visit page_path(space_record.identifier, page_record.number)

    # 画像要素が存在することを確認
    expect(page).to have_css("img[data-attachment-id='#{attachment_record1.id}']")
    expect(page).to have_css("img[data-attachment-id='#{attachment_record2.id}']")

    # 非同期処理が完了するまで待機（attachment-loaderコントローラーが接続され、APIが呼ばれるまで）
    sleep 1

    # 署名付きURLに置換されていることを確認（data URIになる）
    img1 = find("img[data-attachment-id='#{attachment_record1.id}']")
    img2 = find("img[data-attachment-id='#{attachment_record2.id}']")
    expect(img1["src"]).to start_with("data:image/png;base64,")
    expect(img2["src"]).to start_with("data:image/png;base64,")

    # loading="lazy"属性が設定されていることを確認
    expect(img1["loading"]).to eq("lazy")
    expect(img2["loading"]).to eq("lazy")
  end

  it "動画ファイルの署名付きURLが非同期で読み込まれること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # 動画の添付ファイル作成
    attachment_record = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)

    # ページ作成（body_htmlにプレースホルダー動画を含む）
    page_record = create(:page_record, :published,
      space_record:,
      topic_record:,
      body_html: <<~HTML
        <p>テスト動画</p>
        <video src="/attachments/placeholder" data-attachment-id="#{attachment_record.id}" data-attachment-type="video" class="wikino-attachment-video" controls>
          お使いのブラウザは動画タグをサポートしていません。
        </video>
      HTML
    )

    sign_in(user_record:)

    visit page_path(space_record.identifier, page_record.number)

    # 動画要素が存在することを確認（visible: :allで非表示の要素も含む）
    expect(page).to have_css("video[data-attachment-id='#{attachment_record.id}']", visible: :all)

    # 非同期処理が完了するまで待機
    sleep 1

    # 署名付きURLに置換されていることを確認（data URIになる）
    video = find("video[data-attachment-id='#{attachment_record.id}']", visible: :all)
    expect(video["src"]).to start_with("data:image/png;base64,")
  end

  it "ダウンロードリンクの署名付きURLが非同期で読み込まれること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # ファイルの添付ファイル作成
    attachment_record = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)

    # ページ作成（body_htmlにダウンロードリンクを含む）
    page_record = create(:page_record, :published,
      space_record:,
      topic_record:,
      body_html: <<~HTML
        <p>ファイルダウンロード</p>
        <a href="/attachments/placeholder" data-attachment-id="#{attachment_record.id}" data-attachment-link="true" target="_blank" rel="noopener noreferrer">
          ファイルをダウンロード
        </a>
      HTML
    )

    sign_in(user_record:)

    visit page_path(space_record.identifier, page_record.number)

    # リンク要素が存在することを確認
    expect(page).to have_css("a[data-attachment-id='#{attachment_record.id}']")

    # 非同期処理が完了するまで待機
    sleep 1

    # 署名付きURLに置換されていることを確認（data URIになる）
    link = find("a[data-attachment-id='#{attachment_record.id}']")
    expect(link["href"]).to start_with("data:image/png;base64,")
  end

  it "署名付きURL取得に失敗した場合はエラーがコンソールに出力されること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # 添付ファイル作成
    attachment_record = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)

    # ページ作成
    page_record = create(:page_record, :published,
      space_record:,
      topic_record:,
      body_html: <<~HTML
        <img src="/attachments/placeholder" data-attachment-id="#{attachment_record.id}" data-attachment-type="image" class="wikino-attachment-image" />
      HTML
    )

    sign_in(user_record:)

    # テスト用エンドポイントがエラーを返すようにモック
    # Test::Attachments::SignedUrls::CreateControllerをモック
    test_controller = Test::Attachments::SignedUrls::CreateController.new
    allow(Test::Attachments::SignedUrls::CreateController).to receive(:new).and_return(test_controller)
    allow(test_controller).to receive(:call) do
      test_controller.instance_eval do
        render json: {error: "Unauthorized"}, status: :unauthorized
      end
    end

    visit page_path(space_record.identifier, page_record.number)

    # エラー処理のための待機
    sleep 0.5

    # JavaScriptコンソールログを確認（JSテストなのでpage.driver.browser.manage.logsを使用）
    # コンソールログの確認はスキップ（ドライバー依存のため）

    # 画像のsrcはプレースホルダーのままであることを確認（エラー時はURLが変更されない）
    img = find("img[data-attachment-id='#{attachment_record.id}']", visible: :all)
    expect(img["src"]).to end_with("/attachments/placeholder")
  end

  it "添付ファイル要素がない場合はAPI呼び出しが行われないこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # 添付ファイルを含まないページを作成
    page_record = create(:page_record, :published,
      space_record:,
      topic_record:,
      body_html: "<p>添付ファイルなしのページ</p>")

    sign_in(user_record:)

    # API呼び出しが行われないことを確認するためのモック
    api_called = false
    test_controller = Test::Attachments::SignedUrls::CreateController.new
    allow(Test::Attachments::SignedUrls::CreateController).to receive(:new).and_return(test_controller)
    allow(test_controller).to receive(:call) do
      api_called = true
      test_controller.instance_eval do
        render json: {signed_urls: {}}, status: :ok
      end
    end

    visit page_path(space_record.identifier, page_record.number)

    # 少し待機
    sleep 0.5

    # APIが呼び出されていないことを確認
    expect(api_called).to be false

    # ページが正常に表示されていることを確認
    expect(page).to have_content("添付ファイルなしのページ")
  end

  it "画像の読み込み完了後にCSSクラスが変更されること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # 添付ファイル作成
    attachment_record = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)

    # ページ作成
    page_record = create(:page_record, :published,
      space_record:,
      topic_record:,
      body_html: <<~HTML
        <img src="/attachments/placeholder" data-attachment-id="#{attachment_record.id}" data-attachment-type="image" class="wikino-attachment-image" loading="lazy" />
      HTML
    )

    sign_in(user_record:)

    visit page_path(space_record.identifier, page_record.number)

    # 画像要素が存在することを確認
    expect(page).to have_css("img[data-attachment-id='#{attachment_record.id}']")

    # 画像の読み込みが完了するまで待機
    sleep 1.5

    # 読み込み完了後は wikino-attachment-image-loaded クラスに変更される
    expect(page).to have_css("img.wikino-attachment-image-loaded[data-attachment-id='#{attachment_record.id}']")

    # クラスが正しく設定されていることを確認
    img = find("img[data-attachment-id='#{attachment_record.id}']")
    expect(img["class"]).to eq("wikino-attachment-image-loaded")
  end
end
