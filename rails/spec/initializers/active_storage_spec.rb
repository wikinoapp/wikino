# typed: false
# frozen_string_literal: true

# Active Storage runs every variant through libvips because this initializer
# sets `variant_processor = :vips`. Active Storage calls
# `Vips.block_untrusted(true)` while its engine loads, which needs the
# `ruby-vips` binding (2.2.1 or later) and libvips 8.13 or later; the
# application refuses to boot otherwise. These examples pin that precondition
# so that a re-introduced `vips` fork gem, a `ruby-vips` downgrade, or an
# environment carrying an older libvips is caught by CI instead of at deploy
# time.
#
# [Ja] このイニシャライザが `variant_processor = :vips` を設定しているため、
# Active Storage の variant 生成はすべて libvips を通る。Active Storage は
# エンジンのロード時に `Vips.block_untrusted(true)` を呼ぶ。これには
# `ruby-vips` バインディング (2.2.1 以上) と libvips 8.13 以上が必要で、
# 満たさない環境ではアプリが boot しない。`vips` fork gem の再混入・
# `ruby-vips` のダウングレード・古い libvips を持つ環境をデプロイ時ではなく
# CI で検知できるよう、この前提条件を固定する。
RSpec.describe "config/initializers/active_storage.rb" do # rubocop:disable RSpec/DescribeClass
  describe "variant_processor" do
    it "vips を使うこと" do
      expect(Rails.application.config.active_storage.variant_processor).to eq(:vips)
    end
  end

  describe "libvips の untrusted なローダー / セーバー" do
    # A valid 1x1 24-bit BMP: a 14-byte file header, a 40-byte info header, and
    # a single padded BGR pixel. BMP is one of the formats libvips marks as
    # unfuzzed, so it must not load while untrusted operations are blocked.
    #
    # [Ja] 妥当な 1x1 24 bit の BMP。14 byte のファイルヘッダー、40 byte の情報
    # ヘッダー、パディング込みの BGR 1 画素から成る。BMP は libvips が unfuzzed
    # とマークしている形式の 1 つで、untrusted な操作がブロックされている間は
    # 読み込めない。
    let(:bmp_data) do
      file_header = "BM" + [58, 0, 54].pack("l<3")
      info_header = [40, 1, 1].pack("l<3") + [1, 24].pack("s<2") + [0, 4, 0, 0, 0, 0].pack("l<6")
      pixel = [0x00, 0x00, 0xff, 0x00].pack("C4")

      file_header + info_header + pixel
    end

    it "block_untrusted を実装したバインディングが読み込まれていること" do
      expect(Vips).to respond_to(:block_untrusted)
    end

    it "unfuzzed な形式 (BMP) の読み込みを拒否すること" do
      expect { Vips::Image.new_from_buffer(bmp_data, "") }.to raise_error(Vips::Error)
    end

    # Guards the example above from passing for the wrong reason: if the inline
    # BMP were malformed, libvips would reject it whether or not untrusted
    # operations are blocked.
    #
    # [Ja] 上のケースが誤った理由で通ることを防ぐ。インラインの BMP が不正なら、
    # untrusted な操作のブロックの有無によらず libvips は読み込みに失敗する。
    it "ブロックを解除すれば同じデータを読み込めること" do
      width = begin
        Vips.block_untrusted(false)
        Vips::Image.new_from_buffer(bmp_data, "").width
      ensure
        Vips.block_untrusted(true)
      end

      expect(width).to eq(1)
    end

    it "添付ファイルで許可している形式 (PNG) は従来どおり読み込めること" do
      image = Vips::Image.new_from_file(Rails.root.join("spec/fixtures/files/test-image.png").to_s)

      expect(image.width).to be_positive
    end
  end
end
