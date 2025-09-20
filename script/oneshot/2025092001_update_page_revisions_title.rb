# frozen_string_literal: true

# page_revisionsテーブルの既存レコードにtitleを設定するoneshotスクリプト
# 各リビジョンのpage_idから対応するページのtitleを取得して設定する

puts "Starting to update page_revisions title..."

total_count = PageRevisionRecord.count
updated_count = 0
skipped_count = 0

PageRevisionRecord.find_each.with_index do |revision, index|
  # 既にtitleが設定されている場合はスキップ
  if revision.title.present?
    skipped_count += 1
    next
  end

  # 対応するページレコードを取得
  page = revision.page_record

  if page.present?
    # ページのタイトルを設定
    revision.update_column(:title, page.title)
    updated_count += 1
  else
    # ページが見つからない場合（通常はあり得ないが念のため）
    puts "Warning: Page not found for revision #{revision.id}"
    skipped_count += 1
  end

  # 進捗表示（100件ごと）
  if (index + 1) % 100 == 0
    puts "Progress: #{index + 1}/#{total_count} processed..."
  end
end

puts "=" * 50
puts "Update completed!"
puts "Total revisions: #{total_count}"
puts "Updated: #{updated_count}"
puts "Skipped: #{skipped_count}"
